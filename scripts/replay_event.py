#!/usr/bin/env python3
"""Replay exactly one reviewed JSON event without changing its eventId."""
import json,sys,subprocess,uuid
from pathlib import Path
if len(sys.argv)!=2:raise SystemExit('Usage: python3 scripts/replay_event.py event.json')
event=json.loads(Path(sys.argv[1]).read_text())
uuid.UUID(event['eventId'])
if event.get('eventVersion')!=1:raise SystemExit('Only version 1 envelopes are supported')
key=event.get('payload',{}).get('warehouseId',event['eventId'])
subprocess.run(['docker','compose','exec','-T','kafka','/opt/kafka/bin/kafka-console-producer.sh','--bootstrap-server','kafka:9092','--topic','wms.events','--property','parse.key=true'],input=key+'\t'+json.dumps(event)+'\n',text=True,check=True,cwd=Path(__file__).resolve().parents[1])
print('Replayed',event['eventId'])
