import * as gcp from '@pulumi/gcp';
import {
  config,
  createK8sServiceAccountFromGCPServiceAccount,
  createMigrationJob,
  createServiceAccountAndGrantRoles,
  k8sServiceAccountToIdentity,
  getImageTag,
  createKubernetesSecretFromRecord,
  createAutoscaledExposedApplication,
  convertRecordToContainerEnvVars,
  getMemoryAndCpuMetrics,
  createAutoscaledApplication, getPubSubUndeliveredMessagesMetric, getFullSubscriptionLabel, createPubSubCronJobs,
} from '@dailydotdev/pulumi-common';
import { Input } from '@pulumi/pulumi';

const imageTag = getImageTag();
const name = 'monetization';

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

const { namespace } = config.requireObject<{ namespace: string }>('k8s');

const envVars = config.requireObject<Record<string, string>>('env');

const containerEnvVars = convertRecordToContainerEnvVars({
  secretName: name,
  data: envVars,
});

createKubernetesSecretFromRecord({
  data: envVars,
  resourceName: 'k8s-secret',
  name,
  namespace,
});

const image = `gcr.io/daily-ops/daily-${name}:${imageTag}`;

// Create K8S service account and assign it to a GCP service account

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
  containerEnvVars,
  k8sServiceAccount,
);

const limits: Input<{
  [key: string]: Input<string>;
}> = {
  cpu: '1',
  memory: '256Mi',
};

const probe = {
  httpGet: { path: '/health', port: 'http' },
  initialDelaySeconds: 5,
};

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
      readinessProbe: probe,
      livenessProbe: probe,
      env: [
        ...containerEnvVars,
        { name: 'PORT', value: '3000' },
        { name: 'ENV', value: 'PROD' },
      ],
      resources: {
        requests: limits,
        limits,
      },
    },
  ],
  maxReplicas: 10,
  metrics: getMemoryAndCpuMetrics(),
  deploymentDependsOn: [migrationJob],
});

createAutoscaledApplication({
  resourcePrefix: 'bg-',
  name: `${name}-bg`,
  namespace,
  version: imageTag,
  serviceAccount: k8sServiceAccount,
  containers: [
    {
      name: 'app',
      image,
      args: ['/main', 'background'],
      env: containerEnvVars,
      resources: {
        requests: limits,
        limits: limits,
      },
    },
  ],
  minReplicas: 1,
  maxReplicas: 4,
  metrics: [{
    external: {
      metric: {
        name: getPubSubUndeliveredMessagesMetric(),
        selector: {
          matchLabels: {
            [getFullSubscriptionLabel('app')]: name,
          },
        },
      },
      target: {
        type: 'Value',
        averageValue: '20',
      },
    },
    type: 'External',
  }],
  deploymentDependsOn: [migrationJob],
});

const jobs = createPubSubCronJobs(name, [{
  name: 'delete-old-tags',
  schedule: '6 10 * * 0',
  topic: 'delete-old-tags',
}]);

new gcp.pubsub.Subscription(`${name}-sub-delete-old-tags`, {
  topic: 'delete-old-tags',
  name: `${name}-delete-old-tags`,
  labels: { app: name },
  retryPolicy: {
    minimumBackoff: '1s',
    maximumBackoff: '60s',
  },
  expirationPolicy: {
    ttl: '',
  },
}, { dependsOn: jobs });

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
