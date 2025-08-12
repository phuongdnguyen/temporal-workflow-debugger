import {
    exampleWorkflow,
} from './workflow';
import { ReplayMode, replay } from '@phuongdnguyen/replayer-adapter-nodejs';

// ====================
// MAIN EXECUTION
// ====================

async function main() {
    try {
        // Configure adapter for ide replay
        const opts = {
            mode: ReplayMode.IDE,
            workerReplayOptions: {
                workflowsPath: require.resolve('./workflow.ts'),
                bundlerOptions: {
                    ignoreModules: [
                        'fs/promises',
                        '@temporalio/worker',
                        // 'http',
                        'path',
                        'child_process'
                    ]
                },
                debugMode: true,
            },
            debuggerAddr: 'http://127.0.0.1:54578'
        } as any; // adapter types

        await replay(opts, exampleWorkflow);

        console.log('Replay completed successfully');
    } catch (error) {
        console.error('Replay failed:', error);
    }

}

// Run main if this file is executed directly
if (require.main === module) {
    main().catch((error) => {
        console.error('Error:', error);
        process.exit(1);
    });
}
