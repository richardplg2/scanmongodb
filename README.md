# scanmongodb

Fast MongoDB data directory scanner. Finds MongoDB database files (WiredTiger/MMAPv1) for v3/v4/v5.

## Download & Run

**One-liner** — copy paste vào VPS và chạy luôn:

```bash
# AMD64 (hầu hết VPS)
curl -sL https://github.com/richardplg2/scanmongodb/releases/download/v1.0.0/scanmongodb-linux-amd64 -o /tmp/scanmongodb && chmod +x /tmp/scanmongodb

# ARM64 (AWS Graviton, Oracle ARM...)
curl -sL https://github.com/richardplg2/scanmongodb/releases/download/v1.0.0/scanmongodb-linux-arm64 -o /tmp/scanmongodb && chmod +x /tmp/scanmongodb
```

## Usage

```bash
# Scan toàn bộ filesystem (cần sudo để đọc mọi thư mục)
sudo /tmp/scanmongodb /

# Scan 1 path cụ thể
/tmp/scanmongodb /var/lib/mongodb

# Scan nhiều path
/tmp/scanmongodb /data /var/lib /opt

# Output JSON (tiện cho scripting)
/tmp/scanmongodb -json /

# Tuỳ chỉnh số workers (mặc định = CPU * 2)
/tmp/scanmongodb -workers 16 /
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

## Cleanup

```bash
rm /tmp/scanmongodb
```
