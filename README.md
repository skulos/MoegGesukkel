# MoegGesukkel
Ons is Moeg Gesukkel !!

## Start the Server
```
$ go run server/server.go
```

## Start the Client
```
$ go run client/client.go
```

## Create Systemd service file

- 1 Create file at `/lib/systemd/system/`
```
$ nano /lib/systemd/system/moeggesukkel.service
```

- 2 Inside the file
```
[Unit]
Description=Moeggesukkel gRPC Server
ConditionPathExists=/root/moeggesukkel/moeggesukkel_server
After=network.target

[Service]
Type=simple
User=root
Group=root
LimitNOFILE=1024

Restart=on-failure
RestartSec=10
startLimitIntervalSec=60

WorkingDirectory=/root/moeggesukkel
ExecStart=/root/moeggesukkel/moeggesukkel_server

PermissionsStartOnly=true
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=moeggesukkel_server

[Install]
WantedBy=multi-user.target
```