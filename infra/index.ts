import * as gcp from '@pulumi/gcp';
import {
  addIAMRolesToServiceAccount,
  createEnvVarsFromSecret,
  infra,
  location, serviceAccountToMember,
} from './helpers';
import {Output} from '@pulumi/pulumi';

const name = 'monetization';

const vpcConnector = infra.getOutput('serverlessVPC') as Output<gcp.vpcaccess.Connector>;

const serviceAccount = new gcp.serviceaccount.Account(`${name}-sa`, {
  accountId: `daily-${name}`,
  displayName: `daily-${name}`,
});

addIAMRolesToServiceAccount(
  name,
  [
    {name: 'trace', role: 'roles/cloudtrace.agent'},
    {name: 'secret', role: 'roles/secretmanager.secretAccessor'},
  ],
  serviceAccount,
);

const secrets = createEnvVarsFromSecret(name);

const image = `gcr.io/daily-ops/daily-${name}:2604898389313fcb943e5d6fe37177e562b204d5`;

const service = new gcp.cloudrun.Service(name, {
  name,
  location,
  template: {
    metadata: {
      annotations: {
        'autoscaling.knative.dev/maxScale': '20',
        'run.googleapis.com/vpc-access-connector': vpcConnector.name,
      },
    },
    spec: {
      serviceAccountName: serviceAccount.email,
      containers: [
        {
          image,
          resources: {limits: {cpu: '1', memory: '256Mi'}},
          envs: secrets,
        },
      ],
    },
  },
});

const bgService = new gcp.cloudrun.Service(`${name}-background`, {
  name: `${name}-background`,
  location,
  template: {
    metadata: {
      annotations: {
        'autoscaling.knative.dev/maxScale': '20',
        'run.googleapis.com/vpc-access-connector': vpcConnector.name,
      },
    },
    spec: {
      serviceAccountName: serviceAccount.email,
      containers: [
        {
          image,
          resources: {limits: {cpu: '1', memory: '256Mi'}},
          envs: secrets,
          args: ['background'],
        },
      ],
    },
  },
});

new gcp.cloudrun.IamMember(`${name}-public`, {
  service: service.name,
  location,
  role: 'roles/run.invoker',
  member: 'allUsers',
});

export const serviceUrl = service.statuses[0].url;
export const bgServiceUrl = bgService.statuses[0].url;

const cloudRunPubSubInvoker = infra.getOutput('cloudRunPubSubInvoker') as Output<gcp.serviceaccount.Account>;
new gcp.cloudrun.IamMember(`${name}-pubsub-invoker`, {
  service: bgService.name,
  location,
  role: 'roles/run.invoker',
  member: serviceAccountToMember(cloudRunPubSubInvoker)
});

new gcp.pubsub.Subscription(`${name}-sub-segment-found`, {
  topic: 'segment-found',
  name: `${name}-segment-found`,
  pushConfig: {
    pushEndpoint: bgServiceUrl.apply((url) => `${url}/segmentFound`),
    oidcToken: {
      serviceAccountEmail: cloudRunPubSubInvoker.email,
    }
  },
  retryPolicy: {
    minimumBackoff: '10s',
    maximumBackoff: '600s',
  }
});
