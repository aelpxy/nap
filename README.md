# nap (not another platform)

A lightweight self-hosted platform as a service.

## Quick start

### 1. Deploy your first application

```bash
# initialize a project
cd your-app/
nap init

# deploy to nap
nap app deploy myapp .

# your app is now running at http://myapp.nap.local
```

### 2. Create and link a database

```bash
# create a postgresql database
nap db create postgres mydb

# link it to your app (auto-injects DATABASE_URL)
nap app link myapp mydb

# your app automatically restarts with database credentials
```

### 3. Scale your application

```bash
# scale to 3 instances with load balancing
nap app scale myapp --instances 3

# add 2 more instances
nap app scale myapp --add 2

# scale back down
nap app scale myapp --instances 1
```

### 4. Publish with TLS

```bash
# configure publishing settings
nap config setup

# publish your app with automatic https
nap app publish myapp --domain myapp.example.com

# add custom domains
nap app domain add myapp custom.example.com
```

## Usage

### Application management

```bash
# initialize project
nap init                              # interactive setup
nap init --full --name myapp          # full setup with defaults

# deployment
nap app deploy myapp .                # deploy from current directory
nap app deploy myapp ./src --port 8080 --memory 512 --cpu 1.0
nap app deploy myapp . --strategy rolling  # zero-downtime deployment

# application control
nap app list                          # list all applications
nap app status myapp                  # detailed status
nap app logs myapp                    # view logs
nap app logs myapp -f                 # follow logs
nap app restart myapp                 # restart application
nap app stop myapp                    # stop application
nap app start myapp                   # start application
nap app destroy myapp                 # remove application

# scaling
nap app scale myapp --instances 5     # scale to 5 instances
nap app scale myapp --add 2           # add 2 instances
nap app scale myapp --remove 1        # remove 1 instance

# deployment history & rollback
nap app deployments myapp             # view deployment history
nap app rollback myapp                # rollback to previous version
nap app rollback myapp --version 3    # rollback to specific version

# publishing
nap app publish myapp                 # publish with auto-generated domain
nap app publish myapp --domain app.example.com
nap app unpublish myapp               # unpublish (make local-only)
nap app domain add myapp custom.com   # add custom domain
nap app domain remove myapp custom.com
```

### Environment variables

```bash
nap app env set myapp KEY=value KEY2=value2
nap app env list myapp
nap app env import myapp .env
nap app env export myapp > .env
nap app env unset myapp KEY KEY2
```

### Database management

```bash
# create databases
nap db create postgres mydb           # create postgresql
nap db create postgres mydb --password custom-pass
nap db create valkey cache            # create valkey (redis)

# database operations
nap db list                           # list all databases
nap db status mydb                    # detailed status
nap db credentials mydb               # show connection details
nap db logs mydb                      # view logs
nap db logs mydb -f                   # follow logs
nap db start mydb                     # start database
nap db stop mydb                      # stop database
nap db destroy mydb                   # remove database

# publishing (expose to host)
nap db publish mydb --port 5432       # expose on port 5432
nap db unpublish mydb                 # make private

# database linking
nap app link myapp mydb               # link database to app
nap app unlink myapp mydb             # unlink database
nap app databases myapp               # show linked databases
nap db apps mydb                      # show linked apps

# backup and restore
nap db backup mydb                    # create backup
nap db backup mydb --description "before migration"
nap db restore mydb backup-id         # restore from backup
nap db restore mydb backup-id --backup-first
nap db backups list                   # list all backups
nap db backups list mydb              # list backups for specific db
nap db backups delete backup-id       # delete a backup
```

### Volume management

```bash
# backup volumes
nap app volume backup myapp data --description "pre-upgrade"
nap app volume backup myapp data --output /backups/

# restore volumes
nap app volume restore myapp data backup-file.tar.gz

# manage backups
nap app volume backups                # list all volume backups
nap app volume backups myapp          # list for specific app
nap app volume backups myapp data     # list for specific volume
nap app volume backup-delete backup-id
```

### VPC (network) management

```bash
nap vpc create production             # create vpc network
nap vpc list                          # list all vpcs
nap vpc inspect production            # show vpc details
nap vpc delete production             # remove vpc

# deploy to specific vpc
nap app deploy myapp . --vpc production
nap db create postgres mydb --vpc production
```

### Daemon management

```bash
nap daemon status                     # show docker/podman status
nap daemon start                      # start podman (if using podman)
nap daemon stop                       # stop podman
```

### Configuration

```bash
nap config setup                      # configure publishing settings
nap config show                       # display current config
```

## License

MIT License
