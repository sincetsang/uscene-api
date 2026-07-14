pipeline {
    agent {
        kubernetes {
            yaml '''
apiVersion: v1
kind: Pod
metadata:
  labels:
    app: uscene-api-ci
spec:
  containers:
  - name: docker
    image: docker:27-cli
    command:
    - cat
    tty: true
    volumeMounts:
    - name: docker-sock
      mountPath: /var/run/docker.sock
  - name: kubectl
    image: bitnami/kubectl:1.30
    command:
    - cat
    tty: true
    volumeMounts:
    - name: kubeconfig
      mountPath: /kube
  volumes:
  - name: docker-sock
    hostPath:
      path: /var/run/docker.sock
  - name: kubeconfig
    secret:
      secretName: kubeconfig-test
'''
        }
    }

    environment {
        ACR_REGISTRY = 'fsd-registry.ap-southeast-1.cr.aliyuncs.com/uscene'
        IMAGE_TAG = "${BUILD_NUMBER}-${GIT_COMMIT.take(7)}"
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Docker Login') {
            steps {
                container('docker') {
                    withCredentials([usernamePassword(credentialsId: 'acr-creds', usernameVariable: 'ACR_USER', passwordVariable: 'ACR_PASS')]) {
                        sh '''
                            echo "$ACR_PASS" | docker login fsd-registry.ap-southeast-1.cr.aliyuncs.com -u "$ACR_USER" --password-stdin
                        '''
                    }
                }
            }
        }

        stage('Docker Build') {
            steps {
                container('docker') {
                    sh '''
                        docker build -t ${ACR_REGISTRY}/uscene-api:${IMAGE_TAG} .
                        docker tag ${ACR_REGISTRY}/uscene-api:${IMAGE_TAG} ${ACR_REGISTRY}/uscene-api:test-latest
                    '''
                }
            }
        }

        stage('Push To ACR') {
            steps {
                container('docker') {
                    sh '''
                        docker push ${ACR_REGISTRY}/uscene-api:${IMAGE_TAG}
                        docker push ${ACR_REGISTRY}/uscene-api:test-latest
                    '''
                }
            }
        }

        stage('Deploy API') {
            steps {
                container('kubectl') {
                    sh '''
                        kubectl --kubeconfig=/kube/config \
                            set image deployment/uscene-api \
                            api=${ACR_REGISTRY}/uscene-api:${IMAGE_TAG} \
                            -n uscene-test
                    '''
                }
            }
        }

        stage('Deploy Cron') {
            steps {
                container('kubectl') {
                    sh '''
                        kubectl --kubeconfig=/kube/config \
                            set image deployment/uscene-cron \
                            cron=${ACR_REGISTRY}/uscene-api:${IMAGE_TAG} \
                            -n uscene-test
                    '''
                }
            }
        }

        stage('Verify') {
            steps {
                container('kubectl') {
                    sh '''
                        kubectl --kubeconfig=/kube/config rollout status deployment/uscene-api -n uscene-test --timeout=120s
                        kubectl --kubeconfig=/kube/config rollout status deployment/uscene-cron -n uscene-test --timeout=60s
                    '''
                }
            }
        }
    }

    post {
        failure {
            echo "Deploy FAILED: uscene-api ${env.IMAGE_TAG}"
        }
        success {
            echo "Deploy OK: uscene-api ${env.IMAGE_TAG}"
        }
    }
}
