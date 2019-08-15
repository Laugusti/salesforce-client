## sforce rest sobject upsertByExternalId

Create/Update an existing SObject using the Object Name, External ID Field, External ID and data file

### Synopsis

Create/Update an existing SObject using the Object Name, External ID Field, External ID and data file.
With no file or when file is -, read standard input

```
sforce rest sobject upsertByExternalId <name> <extidfield> <extid> [<file>] [flags]
```

### Options

```
  -h, --help   help for upsertByExternalId
```

### Options inherited from parent commands

```
      --config string        config file (default is $HOME/.sforce/config.yml)
      --credentials string   credentials file (default is $HOME/.sforce/credentials.yml)
```

### SEE ALSO

* [sforce rest sobject](sforce_rest_sobject.md)	 - The sobject command performs CRUD operations for Salesforce Objects

###### Auto generated by spf13/cobra on 14-Aug-2019