import { usePaths } from 'vitepress-openapi'
import spec from '../../src/openapi-rpc.json' with {type: 'json'}

export default {
    paths() {
        return usePaths({ spec })
            .getPathsByVerbs()
            .map(({ operationId, summary }) => {
                return {
                    params: {
                        operationId,
                        pageTitle: `${summary} - vitepress-openapi`,
                    },
                }
            })
    },
}