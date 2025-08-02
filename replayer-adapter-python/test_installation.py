#!/usr/bin/env python3
"""
Simple test script to verify the replayer adapter module can be imported correctly.
Run this after installing the package to ensure everything works.
"""

import sys

def test_import():
    """Test that all main components can be imported."""
    try:
        # Test basic import
        import replayer_adapter_python
        print(f"✓ Module imported successfully")
        print(f"  Version: {replayer_adapter_python.__version__}")
        print(f"  Author: {replayer_adapter_python.__author__}")
        
        # Test importing main classes and functions
        from replayer_adapter_python import (
            ReplayMode, ReplayOptions, replay, set_replay_mode, 
            set_breakpoints, RunnerWorkerInterceptor
        )
        print("✓ Main classes and functions imported successfully")
        
        # Test enum values
        print(f"✓ ReplayMode.STANDALONE: {ReplayMode.STANDALONE}")
        print(f"✓ ReplayMode.IDE: {ReplayMode.IDE}")
        
        # Test creating options
        options = ReplayOptions()
        print("✓ ReplayOptions can be instantiated")
        
        # Test creating interceptor
        interceptor = RunnerWorkerInterceptor()
        print("✓ RunnerWorkerInterceptor can be instantiated")
        
        print("\n🎉 All tests passed! The module is properly installed and importable.")
        return True
        
    except ImportError as e:
        print(f"❌ Import failed: {e}")
        print("\nTry installing the module first:")
        print("  pip install -e .")
        return False
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        return False

def main():
    """Main test function."""
    print("Testing replayer-adapter-python module installation...")
    print("=" * 60)
    
    success = test_import()
    
    if not success:
        sys.exit(1)
    
    print("\nFor usage examples, see example.py or the README.md file.")

if __name__ == "__main__":
    main() 