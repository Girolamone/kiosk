import type { CodegenConfig } from '@graphql-codegen/cli'

// The schema is read from the Go server's SDL files rather than from a running
// server. Types can then be regenerated offline, and CI can prove they match
// the schema without standing up a database.
const config: CodegenConfig = {
  schema: '../../apps/api/graph/*.graphqls',
  documents: 'src/operations/*.graphql',
  generates: {
    'src/graphql/generated.ts': {
      plugins: ['typescript', 'typescript-operations', 'typed-document-node'],
      config: {
        // Every operation becomes a TypedDocumentNode, so urql infers the
        // result and variable types with no manual annotation. A field
        // renamed in the Go schema becomes a TypeScript error in both apps.
        useTypeImports: true,
        avoidOptionals: { field: true },
        // A const object rather than a TypeScript enum. Enums emit runtime
        // code, which TypeScript 6's erasableSyntaxOnly rejects, and this
        // form still reads as ProductStatus.Active at the call site.
        enumsAsConst: true,
        scalars: {
          Time: 'string',
          ID: 'string',
        },
      },
    },
  },
}

export default config
