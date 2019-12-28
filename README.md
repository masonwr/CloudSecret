# CloudSecret

Map [GCP Secret Manager](https://cloud.google.com/secret-manager/docs/) secrets to [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/). 

Implemented as a simple CRD. You define CloudSecrets:

```yaml
apiVersion: secrets.masonwr.dev/v1
kind: CloudSecret
metadata:
  name: cloudsecret-sample
spec:
  data:
    SECRET_DATA: projects/<PROJECT_ID>/secrets/test/versions/latest
```

CloudSecret map a key to a Secret Manager Path, and produces a matching Kubernetes secret with the resolved secret data. 

For example, if we applied the above CloudSecret, this would result in the creation of the following Kubernetes secret:

```yaml
apiVersion: v1
data:
  SECRET_DATA: a2VlcCB0...
kind: Secret
```

## Install

> NB: The service account running the deployment must have the "Secret Manager Secret Accessor" role. And the Secret Manager API must abviously be [enabled](https://cloud.google.com/secret-manager/docs/quickstart-secret-manager-console).

### Quick start

```shell
$ git clone https://github.com/masonwr/CloudSecret && cd CloudSecret
$ make install  # install CRD definitionf
$ make deploy   # use public image build from this repo
```

### Custom Image

```shell
$ git clone https://github.com/masonwr/CloudSecret && cd CloudSecret
$ export IMG=your/image/repo:tag
$ make install 
$ make docker-build docker-push
$ make deploy
```



## Tutorial

**Create the GCP Secret, and get its path**

```shell
$ cd $(mktemp -d)
$ export PROJECT_ID=some_project_id
$ echo "keep this secret, keep this safe" > secret.data.txt
$ gcloud beta secrets  create loc-of-ring \
   --data-file=secret.data.txt \
   --project=$PROJECT_ID \
   --replication-policy=automatic
$ gcloud beta secrets describe loc-of-ring --project=$PROJECT_ID
createTime: '2019-12-23T21:11:34.245558Z'
name: projects/<PROJECT_ID>/secrets/test
replication:
  automatic: {}
```

Note the fully qualified secret name.



**Define a CloudSecret**

```shell
$ cat << EOF > cloudSecretExample.yaml
apiVersion: secrets.masonwr.dev/v1                                                                                                   
kind: CloudSecret
metadata:
  name: example
spec:
  data:
    SECRET_DATA: <Fully qulified secret path>/versions/latest
EOF

$ kubectl apply -f cloudSecretExample.yaml 
```



**Verify**

```shell
$ kubectl get secrets example -o json | jq -r .data.SECRET_DATA | base64 -d
keep this secret, keep this safe
```





## Misc

info of polling interval
