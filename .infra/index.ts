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
