#!/usr/bin/env python3
"""Create local credentials once, without overwriting an existing configuration."""
from pathlib import Path
import secrets
root=Path(__file__).resolve().parents[1]
target=root/'.env'
if target.exists():
 print('.env already exists; preserved')
else:
 with target.open('x') as f:
  f.write('POSTGRES_USER=wms\nPOSTGRES_PASSWORD='+secrets.token_hex(24)+'\nGRAFANA_PASSWORD='+secrets.token_hex(24)+'\n')
 target.chmod(0o600)
 print('Created .env. Grafana username: admin; password is in .env.')
