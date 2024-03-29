# Datahub Configuration Deployment: mim-deploy

mim-deploy is a cli to deploy a datahub configuration from a git repo to the Mimiro datahub. It creates a manifest and stores it in the datahub under the content-endpoint and uses this to compare file updates with previous md5 file hashes.
Based on the comparison, it creates a list of operations and utilizes the [mim cli client](https://github.com/mimiro-io/datahub-cli) to execute them.

## Expected configuration file structure
```
├── README.md
├── contents
│   ├── S3
│   │   └── content-s3.json
│   └── mysystem
│       └── content-mysystem.json
├── dataset
│   ├── myfolder
│   │   └── my-dataset.json
│   └── my-other-dataset.json
├── environments
│   ├── variables-dev.json
│   ├── variables-prod.json
│   └── variables-test.json
├── jobs
│   ├── import-mysystem-owner.json
│   ├── import-mysystem-order.json
│   └── order-myothersystem.json
└── transforms
    ├── myTransform.js
    └── order-myothersystem.js
```

## Required configuration changes
* Jobs and content need a type property with either "job" or "content" as value.
* Jobs with transform need to have a "path" property containing the relative path for the transform file inside the transform directory.
```json
{
    "id" : "import-mysystem-owner",
    "type": "job",
    "triggers": [
        {
            "triggerType": "cron",
            "jobType": "incremental",
            "schedule": "@every 120m"
        }
    ],
    "paused": "{{ myVariable }}",
    "source" : {
        "Type" : "HttpDatasetSource",
        "Url" : "http://localhost:4343/datasets/Owner/changes"
    },
    "sink" : {
        "Type" : "DatasetSink",
        "Name": "mysystem.Owner"
    },
    "transform": {
        "Path": "myTransform.js",
        "Type": "JavascriptTransform"
    }
}
```
## Template functionality

### Variables
To use variables in your config files, you can replace a value with `{{ myVariable }}`.
The program will then look for this variable in the  json file defined by the ENVIRONMENT_FILE env variable.

**Example**

ROOT_PATH/environments/variables-dev.json
```json
{
  "myVariable": true
}
```

ROOT_PATH/jobs/import-mysystem-owner.json
```json
{
    "id" : "import-mysystem-owner",
    "type": "job",
    "triggers": [
        {
            "triggerType": "cron",
            "jobType": "incremental",
            "schedule": "@every 120m"
        }
    ],
    "paused": "{{ myVariable }}",
    "source" : {
        "Type" : "HttpDatasetSource",
        "Url" : "http://localhost:4343/datasets/Owner/changes"
    },
    "sink" : {
        "Type" : "DatasetSink",
        "Name": "mysystem.Owner"
    }
}
```

### Include file content
If you have a large configuration file you want to split up into multiple files, you can achieve that by using the include syntax:
```json
{
    "baseNameSpace": "http://data.mimiro.io/mysystem/",
    "baseUri": "http://data.mimiro.io/mysystem/",
    "database": "mydb",
    "id": "db1",
    "type": "content",
    "tableMappings": "{% include list('contents/mysystem/tableMappings/*.json') %}"
}
```
This will then join all matching json files into a list and push the generated json config to the datahub.
If you wish to only add a single object and not a list, you can instead write like this:
```json
{
    "tableMappings": "{% include 'contents/mysystem/tableMappings/owner.json' %}"
}
```
If a wildcard is used in the file path, and it matches more than one file, it will automatically add the content as a list.

### Ignore paths from deployment
To ignore specific paths from being deployed add the environment variable:
```shell
--ignorePath ../datahub-config/<path_to_ignore>
```
to your bash command

### Dataset creation and public namespaces
When you define a `DatasetSink`, the named dataset will be created when the configuration is deployed to the datahub.

**Public namespaces**

If you need to define public namespaces for the dataset used by the sink, this can be defined in the job like this.
```json
    "sink": {
        "Type": "DatasetSink",
        "Name": "mysystem.Owner",
        "publicNamespaces": [
            "http://data.mimiro.io/owner/event/",
            "http://data.mimiro.io/people/birthdate/"
        ]
    }
```
If the dataset is already created, the dataset will be updated with the defined public namespaces.

#### Create dataset and upload entities stored in your config
In some cases we need to datasets that we manually create and can't be read from a different source. This can be achieved by adding files under the `dataset` directory.
Files in there need to structured like this:

```json
{
    "type": "dataset",
    "datasetName": "cima.AnimalType",
    "publicNamespaces": [],
    "entities": [
        {
            "id": "@context",
            "namespaces": {
                "ns1": "http://data.mimiro.io/cima/",
                "ns2": "http://data.mimiro.io/sdb/animaltype/",
                "ns3": "http://www.w3.org/2000/01/rdf-schema#"
            }
        },
        {
            "id": "ns2:cow",
            "refs": {
                "ns3:type": "ns1:AnimalType"
            },
            "props": {
                "ns1:name": "Cow"
            }
        },
        {
            "id": "ns2:pig",
            "refs": {
                "ns3:type": "ns1:AnimalType"
            },
            "props": {
                "ns1:name": "Pig"
            }
        }
    ]
}

```

## How to run

The following configuration properties can either be set by environment variables or by changing the .env file


### Build binary
```shell
make mim-deploy
```

### Deploy to local datahub
```shell
mim-deploy http://localhost:8080 --path ../datahub-config --env ../datahub-config/environments/variables-local.json --dry-run=false
```

### Deploy to remote datahub
```shell
mim login dev --out | mim-deploy https://dev.api.example.com --token-stdin --path ../datahub-config --env ../datahub-config/environments/variables-dev.json --dry-run
```

#### Build docker image
```shell
make docker
```

