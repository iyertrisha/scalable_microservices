pipeline {
  agent any

  environment {
    IMAGE_TAG = "local"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Test') {
      steps {
        sh 'go test ./...'
      }
    }

    stage('Lint') {
      steps {
        sh 'go vet ./...'
      }
    }

    stage('Docker Build') {
      steps {
        sh '''
          docker build --build-arg SERVICE=order-service -t fleetflow/order-service:${IMAGE_TAG} -f deployments/docker/Dockerfile .
          docker build --build-arg SERVICE=driver-service -t fleetflow/driver-service:${IMAGE_TAG} -f deployments/docker/Dockerfile .
          docker build --build-arg SERVICE=dispatch-service -t fleetflow/dispatch-service:${IMAGE_TAG} -f deployments/docker/Dockerfile .
        '''
      }
    }

    stage('Security Scan') {
      steps {
        sh '''
          if command -v trivy >/dev/null 2>&1; then
            trivy image --exit-code 0 fleetflow/order-service:${IMAGE_TAG} || true
          else
            echo "trivy not installed; skipping"
          fi
        '''
      }
    }

    stage('Load into kind') {
      when {
        expression { return fileExists('.kind-enabled') || env.KIND_DEPLOY == 'true' }
      }
      steps {
        sh '''
          kind load docker-image fleetflow/order-service:${IMAGE_TAG} --name fleetflow
          kind load docker-image fleetflow/driver-service:${IMAGE_TAG} --name fleetflow
          kind load docker-image fleetflow/dispatch-service:${IMAGE_TAG} --name fleetflow
          kubectl apply -k deployments/k8s/overlays/local
        '''
      }
    }
  }
}
