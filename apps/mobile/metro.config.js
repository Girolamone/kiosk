const { getDefaultConfig } = require('expo/metro-config')
const path = require('node:path')

const projectRoot = __dirname
const workspaceRoot = path.resolve(projectRoot, '../..')

const config = getDefaultConfig(projectRoot)

// Metro does not understand npm workspaces on its own: it only watches the
// app folder, so an edit in packages/shared would not trigger a reload and
// an import of it would not resolve.
config.watchFolders = [workspaceRoot]
config.resolver.nodeModulesPaths = [
  path.resolve(projectRoot, 'node_modules'),
  path.resolve(workspaceRoot, 'node_modules'),
]
// Without this, a package hoisted to the workspace root can be resolved
// twice and React ends up loaded in two copies, which fails at runtime with
// errors that point nowhere near the cause.
config.resolver.disableHierarchicalLookup = true

module.exports = config
