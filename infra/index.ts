import * as gcp from '@pulumi/gcp';
import {
  CloudRunAccess,
  config,
  createCloudRunService,
  createEnvVarsFromSecret,
  createK8sServiceAccountFromGCPServiceAccount,
  createMigrationJob,
  createServiceAccountAndGrantRoles,
  getCloudRunPubSubInvoker,
  infra,
  k8sServiceAccountToIdentity,
  createCronJobs,
  getImageTag,
  createKubernetesSecretFromRecord,
  createAutoscaledExposedApplication, convertRecordToContainerEnvVars, getMemoryAndCpuMetrics,
} from '@dailydotdev/pulumi-common';
import { Input, Output } from '@pulumi/pulumi';

const imageTag = getImageTag();
const name = 'monetization';

const vpcConnector = infra.getOutput('serverlessVPC') as Output<gcp.vpcaccess.Connector>;

const { serviceAccount } = createServiceAccountAndGrantRoles(
  `${name}-sa`,
  name,
  `daily-${name}`,
  [
    { name: 'trace', role: 'roles/cloudtrace.agent' },
    { name: 'secret', role: 'roles/secretmanager.secretAccessor' },
    { name: 'subscriber', role: 'roles/pubsub.subscriber' },
  ],
);

const secrets = createEnvVarsFromSecret(name);

const image = `gcr.io/daily-ops/daily-${name}:${imageTag}`;

// Create K8S service account and assign it to a GCP service account
const { namespace } = config.requireObject<{ namespace: string }>('k8s');

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

const limits: Input<{
  [key: string]: Input<string>;
}> = {
  cpu: '1',
  memory: '256Mi',
};

// Deploy to Cloud Run (foreground & background)
const service = createCloudRunService(
  name,
  image,
  secrets,
  limits,
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
  { cpu: '1', memory: '256Mi' },
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

const envVars = config.requireObject<Record<string, string>>('env');

createKubernetesSecretFromRecord({
  data: envVars,
  resourceName: 'k8s-secret',
  name,
  namespace,
});

createAutoscaledExposedApplication({
  name,
  namespace: namespace,
  version: imageTag,
  serviceAccount: k8sServiceAccount,
  containers: [
    {
      name: 'app',
      image,
      ports: [{ name: 'http', containerPort: 3000, protocol: 'TCP' }],
      readinessProbe: {
        httpGet: { path: '/health', port: 'http' },
      },
      env: [
        ...convertRecordToContainerEnvVars({ secretName: name, data: envVars }),
        { name: 'PORT', value: '3000' },
      ],
      resources: {
        requests: limits,
        limits,
      },
    },
  ],
  maxReplicas: 10,
  metrics: getMemoryAndCpuMetrics(),
});

new gcp.pubsub.Subscription(`${name}-sub-views`, {
  topic: 'views',
  name: `${name}-views`,
  labels: { app: name },
  retryPolicy: {
    minimumBackoff: '1s',
    maximumBackoff: '60s',
  },
});

new gcp.pubsub.Subscription(`${name}-sub-new-ad`, {
  topic: 'ad-image-processed',
  name: `${name}-new-ad`,
  labels: { app: name },
  retryPolicy: {
    minimumBackoff: '1s',
    maximumBackoff: '60s',
  },
  expirationPolicy: {
    ttl: '',
  },
});

createCronJobs(name, [{
  name: 'delete-old-tags',
  schedule: '6 10 * * 0',
}], bgServiceUrl);
