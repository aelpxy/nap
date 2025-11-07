# yap (yet another platform)

A lightweight self-hosted platform as a service.

## Quick start

### 1. Deploy your first application

```bash
# initialize a project
cd your-app/
yap init

# deploy to yap
yap app deploy myapp .

# your app is now running at http://myapp.yap.local
```

### 2. Create and link a database

```bash
# create a postgresql database
yap db create postgres mydb

# link it to your app (auto-injects DATABASE_URL)
yap app link myapp mydb

# your app automatically restarts with database credentials
```

### 3. Scale your application

```bash
# scale to 3 instances with load balancing
yap app scale myapp --instances 3

# add 2 more instances
yap app scale myapp --add 2

# scale back down
yap app scale myapp --instances 1
```

### 4. Publish with TLS

```bash
# configure publishing settings
yap config setup

# publish your app with automatic https
yap app publish myapp --domain myapp.example.com

# add custom domains
yap app domain add myapp custom.example.com
```

## Usage

### Application management

```bash
# initialize project
yap init                              # interactive setup
yap init --full --name myapp          # full setup with defaults

# deployment
yap app deploy myapp .                # deploy from current directory
yap app deploy myapp ./src --port 8080 --memory 512 --cpu 1.0
yap app deploy myapp . --strategy rolling  # zero-downtime deployment

# application control
yap app list                          # list all applications
yap app status myapp                  # detailed status
yap app logs myapp                    # view logs
yap app logs myapp -f                 # follow logs
yap app restart myapp                 # restart application
yap app stop myapp                    # stop application
yap app start myapp                   # start application
yap app destroy myapp                 # remove application

# scaling
yap app scale myapp --instances 5     # scale to 5 instances
yap app scale myapp --add 2           # add 2 instances
yap app scale myapp --remove 1        # remove 1 instance

# deployment history & rollback
yap app deployments myapp             # view deployment history
yap app rollback myapp                # rollback to previous version
yap app rollback myapp --version 3    # rollback to specific version

# publishing
yap app publish myapp                 # publish with auto-generated domain
yap app publish myapp --domain app.example.com
yap app unpublish myapp               # unpublish (make local-only)
yap app domain add myapp custom.com   # add custom domain
yap app domain remove myapp custom.com
```

### Environment variables

```bash
yap app env set myapp KEY=value KEY2=value2
yap app env list myapp
yap app env import myapp .env
yap app env export myapp > .env
yap app env unset myapp KEY KEY2
```

### Database management

```bash
# create databases
yap db create postgres mydb           # create postgresql
yap db create postgres mydb --password custom-pass
yap db create valkey cache            # create valkey (redis)

# database operations
yap db list                           # list all databases
yap db status mydb                    # detailed status
yap db credentials mydb               # show connection details
yap db logs mydb                      # view logs
yap db logs mydb -f                   # follow logs
yap db start mydb                     # start database
yap db stop mydb                      # stop database
yap db destroy mydb                   # remove database

# publishing (expose to host)
yap db publish mydb --port 5432       # expose on port 5432
yap db unpublish mydb                 # make private

# database linking
yap app link myapp mydb               # link database to app
yap app unlink myapp mydb             # unlink database
yap app databases myapp               # show linked databases
yap db apps mydb                      # show linked apps

# backup and restore
yap db backup mydb                    # create backup
yap db backup mydb --description "before migration"
yap db restore mydb backup-id         # restore from backup
yap db restore mydb backup-id --backup-first
yap db backups list                   # list all backups
yap db backups list mydb              # list backups for specific db
yap db backups delete backup-id       # delete a backup
```

### Volume management

```bash
# backup volumes
yap app volume backup myapp data --description "pre-upgrade"
yap app volume backup myapp data --output /backups/

# restore volumes
yap app volume restore myapp data backup-file.tar.gz

# manage backups
yap app volume backups                # list all volume backups
yap app volume backups myapp          # list for specific app
yap app volume backups myapp data     # list for specific volume
yap app volume backup-delete backup-id
```

### VPC (network) management

```bash
yap vpc create production             # create vpc network
yap vpc list                          # list all vpcs
yap vpc inspect production            # show vpc details
yap vpc delete production             # remove vpc

# deploy to specific vpc
yap app deploy myapp . --vpc production
yap db create postgres mydb --vpc production
```

### Daemon management

```bash
yap daemon status                     # show docker/podman status
yap daemon start                      # start podman (if using podman)
yap daemon stop                       # stop podman
```

### Configuration

```bash
yap config setup                      # configure publishing settings
yap config show                       # display current config
```

## License

MIT License
