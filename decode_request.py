#!/usr/bin/env python3
# Decode URL-encoded variables từ request để hiểu cấu trúc

import urllib.parse
import json
import sys

# Read the request file
with open(r'e:\WEMAKE\DocWeMake\FlowRegFB_IOS\Login_IOSMes\[179] request_graph.facebook.com_message.txt', 'r', encoding='utf-8') as f:
    content = f.read()

# Find variables parameter
import re
match = re.search(r'variables=([^&]+)', content)
if match:
    encoded = match.group(1)
    decoded = urllib.parse.unquote(encoded)
    
    # Parse as JSON
    try:
        data = json.loads(decoded)
        
        # Pretty print with indentation
        print("=== DECODED VARIABLES ===")
        print(json.dumps(data, indent=2, ensure_ascii=False))
        
        # Check params structure
        if 'params' in data:
            params = data['params']
            if 'params' in params:
                print("\n=== PARAMS.PARAMS (escaped JSON string) ===")
                print(f"Type: {type(params['params'])}")
                print(f"First 500 chars: {params['params'][:500]}")
                
                # Try to decode it
                try:
                    # It's a JSON string, decode it
                    params_inner = json.loads(params['params'])
                    print("\n=== DECODED PARAMS.PARAMS ===")
                    print(json.dumps(params_inner, indent=2, ensure_ascii=False)[:2000])
                except:
                    print("Could not decode params.params as JSON")
    except Exception as e:
        print(f"Error: {e}")
        print(f"Decoded (first 1000 chars): {decoded[:1000]}")
