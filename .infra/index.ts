import * as gcp from '@pulumi/gcp';
import {
    config,
    createServiceAccountAndGrantRoles,
    getImageTag,
    createPubSubCronJobs,
    deployApplicationSuite
} from '@dailydotdev/pulumi-common';
import {Input} from '@pulumi/pulumi';

const imageTag = getImageTag();
const name = 'monetization';

const {serviceAccount} = createServiceAccountAndGrantRoles(
    `${name}-sa`,
    name,
    `daily-${name}`,
    [
        {name: 'trace', role: 'roles/cloudtrace.agent'},
        {name: 'secret', role: 'roles/secretmanager.secretAccessor'},
        {name: 'subscriber', role: 'roles/pubsub.subscriber'},
    ],
);

const {namespace} = config.requireObject<{ namespace: string }>('k8s');

const envVars = config.requireObject<Record<string, string>>('env');

const image = `gcr.io/daily-ops/daily-${name}:${imageTag}`;

const apiLimits: Input<{
    [key: string]: Input<string>;
}> = {
    cpu: '1',
    memory: '256Mi',
};

const bgLimits: Input<{
    [key: string]: Input<string>;
}> = {
    cpu: '500m',
    memory: '256Mi',
};

const probe = {
    httpGet: {path: '/health', port: 'http'},
    initialDelaySeconds: 5,
};

// const deployKubernetesResources = (name: string, isPrimary: boolean, {
//     provider,
//     resourcePrefix = '',
// }: { provider?: ProviderResource; resourcePrefix?: string } = {}): void => {
//     createKubernetesSecretFromRecord({
//         data: envVars,
//         resourceName: `${resourcePrefix}k8s-secret`,
//         name,
//         namespace,
//         provider,
//     });
//     // Create K8S service account and assign it to a GCP service account
//     const k8sServiceAccount = createK8sServiceAccountFromGCPServiceAccount(
//         `${resourcePrefix}${name}-k8s-sa`,
//         name,
//         namespace,
//         serviceAccount,
//         provider
//     );
//     new gcp.serviceaccount.IAMBinding(`${resourcePrefix}${name}-k8s-iam-binding`, {
//         role: 'roles/iam.workloadIdentityUser',
//         serviceAccountId: serviceAccount.id,
//         members: [k8sServiceAccountToIdentity(k8sServiceAccount)],
//     });
//
//     const deploymentDependsOn: Input<Resource>[] = [];
//     if (isPrimary) {
//         const migrationJob = createMigrationJob(
//             `${name}-migration`,
//             namespace,
//             image,
//             ['/main', 'migrate'],
//             containerEnvVars,
//             k8sServiceAccount,
//             {provider, resourcePrefix},
//         );
//         deploymentDependsOn.push(migrationJob);
//     }
//
//     createAutoscaledApplication({
//         resourcePrefix: `${resourcePrefix}bg-`,
//         name: `${name}-bg`,
//         namespace,
//         version: imageTag,
//         serviceAccount: k8sServiceAccount,
//         containers: [
//             {
//                 name: 'app',
//                 image,
//                 args: ['/main', 'background'],
//                 env: containerEnvVars,
//                 resources: {
//                     requests: bgLimits,
//                     limits: bgLimits,
//                 },
//             },
//         ],
//         minReplicas: 1,
//         maxReplicas: 4,
//         metrics: [{
//             external: {
//                 metric: {
//                     name: getPubSubUndeliveredMessagesMetric(),
//                     selector: {
//                         matchLabels: {
//                             [getFullSubscriptionLabel('app')]: name,
//                         },
//                     },
//                 },
//                 target: {
//                     type: 'Value',
//                     averageValue: '20',
//                 },
//             },
//             type: 'External',
//         }],
//         deploymentDependsOn,
//         provider,
//     });
//
//     createAutoscaledExposedApplication({
//         resourcePrefix,
//         name,
//         namespace: namespace,
//         version: imageTag,
//         serviceAccount: k8sServiceAccount,
//         containers: [
//             {
//                 name: 'app',
//                 image,
//                 ports: [{name: 'http', containerPort: 3000, protocol: 'TCP'}],
//                 readinessProbe: probe,
//                 livenessProbe: probe,
//                 env: [
//                     ...containerEnvVars,
//                     {name: 'PORT', value: '3000'},
//                     {name: 'ENV', value: 'PROD'},
//                 ],
//                 resources: {
//                     requests: apiLimits,
//                     limits: apiLimits,
//                 },
//                 lifecycle: gracefulTerminationHook(),
//             },
//         ],
//         maxReplicas: 10,
//         metrics: getMemoryAndCpuMetrics(),
//         deploymentDependsOn,
//         provider,
//     });
// }

const jobs = createPubSubCronJobs(name, [{
    name: 'delete-old-tags',
    schedule: '6 10 * * 0',
    topic: 'delete-old-tags',
}]);

new gcp.pubsub.Subscription(`${name}-sub-delete-old-tags`, {
    topic: 'delete-old-tags',
    name: `${name}-delete-old-tags`,
    labels: {app: name},
    retryPolicy: {
        minimumBackoff: '1s',
        maximumBackoff: '60s',
    },
    expirationPolicy: {
        ttl: '',
    },
}, {dependsOn: jobs});

new gcp.pubsub.Subscription(`${name}-sub-views`, {
    topic: 'views',
    name: `${name}-views`,
    labels: {app: name},
    retryPolicy: {
        minimumBackoff: '1s',
        maximumBackoff: '60s',
    },
});

new gcp.pubsub.Subscription(`${name}-sub-new-ad`, {
    topic: 'ad-image-processed',
    name: `${name}-new-ad`,
    labels: {app: name},
    retryPolicy: {
        minimumBackoff: '1s',
        maximumBackoff: '60s',
    },
    expirationPolicy: {
        ttl: '',
    },
});

// const vpcNativeProvider = getVpcNativeCluster();
// deployKubernetesResources(name, true);
// deployKubernetesResources(name, false, {provider: vpcNativeProvider.provider, resourcePrefix: 'vpc-native-'});

deployApplicationSuite({
    name,
    namespace,
    image,
    imageTag,
    serviceAccount,
    secrets: envVars,
    migration: {
        args: ['/main', 'migrate']
    },
    apps: [{
        port: 3000,
        env: [{name: 'PORT', value: '3000'}, {name: 'ENV', value: 'PROD'}],
        maxReplicas: 10,
        limits: apiLimits,
        readinessProbe: probe,
        metric: {type: 'memory_cpu', cpu: 70},
        createService: true,
    }, {
        nameSuffix: 'bg',
        args: ['/main', 'background'],
        minReplicas: 1,
        maxReplicas: 4,
        limits: bgLimits,
        metric: {type: 'pubsub', labels: {app: name}, targetAverageValue: 20},
    }],
});
