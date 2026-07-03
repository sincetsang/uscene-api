pipeline {
    agent any

    environment {
        ACR_REGISTRY = 'registry.cn-hangzhou.aliyuncs.com/uscene'
        ACR_CREDS   = credentials('acr-creds')
        KUBECONFIG  = credentials('kubeconfig-test')
        IMAGE_TAG   = "${BUILD_NUMBER}-${GIT_COMMIT.take(7)}"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Docker Build') {
            steps {
                sh "docker build -t ${ACR_REGISTRY}/uscene-api:${IMAGE_TAG} ."
                sh "docker tag ${ACR_REGISTRY}/uscene-api:${IMAGE_TAG} ${ACR_REGISTRY}/uscene-api:test-latest"
            }
        }

        stage('Push to ACR') {
            steps {
                sh "docker login -u ${ACR_CREDS_USR} -p ${ACR_CREDS_PSW} ${ACR_REGISTRY}"
                sh "docker push ${ACR_REGISTRY}/uscene-api:${IMAGE_TAG}"
                sh "docker push ${ACR_REGISTRY}/uscene-api:test-latest"
            }
        }

        stage('Deploy API') {
            steps {
                sh '''
                    kubectl --kubeconfig=${KUBECONFIG} \
                        set image deployment/uscene-api \
                        api=${ACR_REGISTRY}/uscene-api:${IMAGE_TAG} \
                        -n uscene-test
                '''
            }
        }

        stage('Deploy Cron') {
            steps {
                sh '''
                    kubectl --kubeconfig=${KUBECONFIG} \
                        set image deployment/uscene-cron \
                        cron=${ACR_REGISTRY}/uscene-api:${IMAGE_TAG} \
                        -n uscene-test
                '''
            }
        }

        stage('Verify') {
            steps {
                sh '''
                    kubectl --kubeconfig=${KUBECONFIG} \
                        rollout status deployment/uscene-api \
                        -n uscene-test --timeout=120s
                    kubectl --kubeconfig=${KUBECONFIG} \
                        rollout status deployment/uscene-cron \
                        -n uscene-test --timeout=60s
                '''
            }
        }
    }

    post {
        failure {
            echo "Deploy FAILED: uscene-api ${IMAGE_TAG}"
        }
        success {
            echo "Deploy OK: uscene-api ${IMAGE_TAG}"
        }
    }
}
