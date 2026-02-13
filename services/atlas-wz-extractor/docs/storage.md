# Storage

This service does not use a database. All output is written to the filesystem.

## Output Directories

### XML Output (`OUTPUT_XML_DIR`)

```
{tenantId}/{region}/{majorVersion}.{minorVersion}/{wzName}.wz/{dirPath}/{imageName}.img.xml
```

### Icon Output (`OUTPUT_IMG_DIR`)

```
{tenantId}/{region}/{majorVersion}.{minorVersion}/{category}/{entityId}/icon.png
```

Categories: `npc`, `mob`, `reactor`, `item`, `skill`.
