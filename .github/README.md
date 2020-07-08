# GitHub Actions
We deploy Atlas Broker to Cloud Foundry using [GitHub Actions](https://docs.github.com/en/actions)

# Folder structure description

## action
The folder consists of implemented GitHub actions which are used in "workflows".
Actions available right now:
- cleanup-cf - clean CF from the previous deployment
- prepare-cf - deploy atlas broker with "default" scenario: pass APIKeys along with `create-service-broker` command
- prepare-credhub - deploy atlas broker with credhub multikeys scenario
Planned actions:
- prepare-template - deploy atlas broker with provided templates
!NOTE! Only Dynamic Plans are currently supported (right?)

"prepare-##" actions have commented parts in case we need spring-music as a test application 

## base-dockerfile
The Dockerfile included here is used for actions and also contains helper functions for actions

## disabled-workflows
All disabled/cancelled/saved for future use workflows are placed in here. For example, workflow "Deploy to Amazon ECS" we plan to use it later.

## workflows
Active workflows for operating. 
- `clean-cf.yml` clean and prepare cloud foundry
- `deploy-broker.yaml` deploy broker to CF

# Using GitHub Actions locally
Tools for successfully running pipeline locally:
- `act` [allow run GitHub action without pushing changes to repo](https://github.com/nektos/act)
- `githubsecrets` allow change/create repository [Github secrets](https://github.com/unfor19/githubsecrets)

## Req. for running act
Put the file `.actrc` to the root project folder with used secrets in GitHub

```
-s ATLAS_BROKER_URL=<url to already deployed broker, used for Deploy to Amazon ECS>
-s ATLAS_PRIVATE_KEY=<private key>
-s ATLAS_PROJECT_ID=<first project id>
-s ATLAS_PROJECT_ID_BAY=<second project id>
-s ATLAS_PUBLIC_KEY=<public key>
-s PCF_PASSWORD=<password>
-s PCF_URL=https://pcf.something.com
-s PCF_USER=<user>
-s CREDHUB_FILE=<file sample>
```

Now simply call:
```
act delete #call clean-cf workflow
act push #call deploy-broker
act <trigger>
```

!NOTE! deploy-broker workflow deletes installed services at the end. If you need to look at prepared services - uncomment/delete "Cleanup ENV for current branch" step from deploy-broker.yml
