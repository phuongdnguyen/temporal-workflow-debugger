#!/usr/bin/env python3
"""Setup script for the Python Replayer Adapter for Temporal."""

from setuptools import setup, find_packages
import os

# Read the README file for long description
def read_readme():
    readme_path = os.path.join(os.path.dirname(__file__), "README.md")
    if os.path.exists(readme_path):
        with open(readme_path, "r", encoding="utf-8") as f:
            return f.read()
    return ""

# Read requirements from requirements.txt
def read_requirements():
    req_path = os.path.join(os.path.dirname(__file__), "requirements.txt")
    if os.path.exists(req_path):
        with open(req_path, "r", encoding="utf-8") as f:
            return [line.strip() for line in f if line.strip() and not line.startswith("#")]
    return []

setup(
    name="temporal-replayer-adapter-python",
    version="0.1.0",
    description="Python Replayer Adapter for Temporal workflows with debugging capabilities",
    long_description=read_readme(),
    long_description_content_type="text/markdown",
    author="Temporal Technologies",
    author_email="support@temporal.io",
    url="https://github.com/temporalio/temporal-goland-plugin",
    packages=["replayer_adapter_python"],
    package_dir={"replayer_adapter_python": "."},
    install_requires=read_requirements(),
    classifiers=[
        "Development Status :: 3 - Alpha",
        "Intended Audience :: Developers",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: Software Development :: Debuggers",
        "Topic :: Software Development :: Libraries :: Python Modules",
    ],
    python_requires=">=3.8",
    keywords="temporal workflow debugging replay interceptor",
    project_urls={
        "Bug Reports": "https://github.com/temporalio/temporal-goland-plugin/issues",
        "Source": "https://github.com/temporalio/temporal-goland-plugin",
        "Documentation": "https://github.com/temporalio/temporal-goland-plugin/blob/main/replayer-adapter-python/README.md",
    },
    include_package_data=True,
    zip_safe=False,
) 