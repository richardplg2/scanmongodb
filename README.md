# scanmongodb

Fast MongoDB data directory scanner. Finds MongoDB database files (WiredTiger/MMAPv1) across your filesystem.

## VPS One-liner

```bash
# AMD64
curl -sL https://github.com/YOUR_USER/scanmongodb/releases/latest/download/scanmongodb-linux-amd64 -o /tmp/scanmongodb && chmod +x /tmp/scanmongodb && /tmp/scanmongodb /

# ARM64
curl -sL https://github.com/YOUR_USER/scanmongodb/releases/latest/download/scanmongodb-linux-arm64 -o /tmp/scanmongodb && chmod +x /tmp/scanmongodb && /tmp/scanmongodb /
```

## Usage

```bash
# Scan specific path
./scanmongodb /var/lib/mongodb

# Scan entire filesystem
sudo ./scanmongodb /

# JSON output (for scripting)
./scanmongodb -json /data

# Multiple paths
./scanmongodb /data /var/lib /opt
```

## Build

```bash
# Local
go build -o scanmongodb .

# Cross-compile for Linux
make build-linux
```

## Output

```
Found 1 MongoDB data directory(ies):

  Path:      /var/lib/mongodb
  Engine:    WiredTiger
  Version:   WiredTiger 10.0.2: (October 25, 2022)
  Size:      1542 MB
  Journal:   true
  Databases: admin, config, myapp

Scanned 84521 dirs in 1.2s
```
