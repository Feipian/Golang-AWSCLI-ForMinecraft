
```
You have to create one key pair for your ec2 instance, for ssh login
```

# Use the downloaded .pem file
ssh -i "C:\path\to\minecraft-key.pem" ec2-user@PUBLIC_IP

```
Replace the path and publicIP
```


# Cleanup.go
It can clean all resource our create, avoid forget close ec2 to getting charge

- [X] VPC
- [X] Internet GateWay


- [ ] bash for install paper mc
- [ ] how to load plugin to my minecraft server using s3??
    - [ ] or using docker package my paper mc using some special feature run server?? 