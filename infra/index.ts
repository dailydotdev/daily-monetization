import * as gcp from '@pulumi/gcp';
import {
    CloudRunAccess,
    config, createCloudRunService, createEnvVarsFromSecret,
    createK8sServiceAccountFromGCPServiceAccount, createMigrationJob,
    createServiceAccountAndGrantRoles, getCloudRunPubSubInvoker,
    imageTag, infra, k8sServiceAccountToIdentity, createCronJobs
} from '@dailydotdev/pulumi-common';
import {Output} from '@pulumi/pulumi';

const name = 'monetization';

const vpcConnector = infra.getOutput('serverlessVPC') as Output<gcp.vpcaccess.Connector>;

const {serviceAccount} = createServiceAccountAndGrantRoles(
    `${name}-sa`,
    name,
    `daily-${name}`,
    [
        {name: 'trace', role: 'roles/cloudtrace.agent'},
        {name: 'secret', role: 'roles/secretmanager.secretAccessor'},
    ],
);

const secrets = createEnvVarsFromSecret(name);

const image = `gcr.io/daily-ops/daily-${name}:${imageTag}`;

// Create K8S service account and assign it to a GCP service account
const {namespace} = config.requireObject<{ namespace: string }>('k8s');

const k8sServiceAccount = createK8sServiceAccountFromGCPServiceAccount(
    `${name}-k8s-sa`,
    name,
    namespace,
    serviceAccount,
);

new gcp.serviceaccount.IAMBinding(`${name}-k8s-iam-binding`, {
    role: 'roles/iam.workloadIdentityUser',
    serviceAccountId: serviceAccount.id,
    members: [k8sServiceAccountToIdentity(k8sServiceAccount)],
});

const migrationJob = createMigrationJob(
    `${name}-migration`,
    namespace,
    image,
    ['/main', 'migrate'],
    secrets,
    k8sServiceAccount,
);

// Deploy to Cloud Run (foreground & background)
const service = createCloudRunService(
    name,
    image,
    secrets,
    {cpu: '1', memory: '256Mi'},
    vpcConnector,
    serviceAccount,
    {
        minScale: 1,
        concurrency: 250,
        dependsOn: [migrationJob],
        access: CloudRunAccess.Public,
        iamMemberName: `${name}-public`,
    },
);

const bgService = createCloudRunService(
    `${name}-background`,
    image,
    secrets,
    {cpu: '1', memory: '256Mi'},
    vpcConnector,
    serviceAccount,
    {
        dependsOn: [migrationJob],
        access: CloudRunAccess.PubSub,
        iamMemberName: `${name}-pubsub-invoker`,
        args: ['background'],
    },
);

export const serviceUrl = service.statuses[0].url;
export const bgServiceUrl = bgService.statuses[0].url;

const cloudRunPubSubInvoker = getCloudRunPubSubInvoker();

new gcp.pubsub.Subscription(`${name}-sub-views`, {
    topic: 'views',
    name: `${name}-views`,
    pushConfig: {
        pushEndpoint: bgServiceUrl.apply((url) => `${url}/view`),
        oidcToken: {
            serviceAccountEmail: cloudRunPubSubInvoker.email,
        }
    },
    retryPolicy: {
        minimumBackoff: '10s',
        maximumBackoff: '600s',
    }
});

new gcp.pubsub.Subscription(`${name}-sub-new-ad`, {
    topic: 'ad-image-processed',
    name: `${name}-new-ad`,
    pushConfig: {
        pushEndpoint: bgServiceUrl.apply((url) => `${url}/newAd`),
        oidcToken: {
            serviceAccountEmail: cloudRunPubSubInvoker.email,
        }
    },
    retryPolicy: {
        minimumBackoff: '10s',
        maximumBackoff: '600s',
    },
    expirationPolicy: {
        ttl: '',
    },
});

createCronJobs(name, [{
    name: 'delete-old-tags',
    schedule: '6 10 * * 0',
}], bgServiceUrl);
