#!/usr/bin/env python3
"""TarsEnterprise Python Adapter — Environment Detection & Tool Listing"""
import sys
import subprocess
import json

def detect():
    result = {
        "lang": "python",
        "version": f"{sys.version_info.major}.{sys.version_info.minor}.{sys.version_info.micro}",
        "interpreter": sys.executable,
        "packages": [],
        "tools": []
    }
    try:
        import pkg_resources
        result["packages"] = [d.project_name for d in pkg_resources.working_set]
    except Exception:
        pass
    # Check common tools
    tools = ["pip", "pytest", "black", "flake8", "mypy", "jupyter"]
    for tool in tools:
        try:
            subprocess.run([tool, "--version"], capture_output=True, check=True)
            result["tools"].append(tool)
        except Exception:
            pass
    return result

if __name__ == "__main__":
    print(json.dumps(detect(), indent=2))
