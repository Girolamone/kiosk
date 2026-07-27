import { useState } from 'react'
import {
  ActivityIndicator,
  FlatList,
  Image,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native'
import { useMutation, useQuery } from 'urql'
import {
  MyStoresDocument,
  ProductStatus,
  StoreProductsDocument,
  UpdateProductDocument,
  formatMoney,
  type ProductCardFragment,
} from '@kiosk/shared'
import { Button, Centered, Notice } from '../components'
import { colors, radius, spacing } from '../theme'
import { AddProduct } from './AddProduct'

export function Shop({ onSignOut }: { onSignOut: () => void }) {
  const [adding, setAdding] = useState(false)

  const [{ data: storesData, fetching: loadingStores, error: storesError }] = useQuery({
    query: MyStoresDocument,
  })
  const store = storesData?.myStores[0]

  const [{ data, fetching }, refetch] = useQuery({
    query: StoreProductsDocument,
    variables: { slug: store?.slug ?? '' },
    pause: !store,
  })

  const [{ fetching: updating }, updateProduct] = useMutation(UpdateProductDocument)
  // Which row is mid-flight, so only that button shows a spinner rather than
  // every row going busy at once.
  const [changing, setChanging] = useState<string | null>(null)

  const reload = () => refetch({ requestPolicy: 'network-only' })

  async function togglePublished(product: ProductCardFragment) {
    setChanging(product.id)
    const next =
      product.status === ProductStatus.Active ? ProductStatus.Draft : ProductStatus.Active
    const result = await updateProduct({ input: { id: product.id, status: next } })
    setChanging(null)
    if (!result.error) reload()
  }

  if (loadingStores) {
    return (
      <Centered>
        <Text style={styles.muted}>Loading…</Text>
      </Centered>
    )
  }

  if (storesError) {
    return (
      <Centered>
        <Notice tone="error">{storesError.message.replace(/^\[GraphQL\]\s*/, '')}</Notice>
        <Button label="Sign out" variant="secondary" onPress={onSignOut} />
      </Centered>
    )
  }

  if (!store) {
    return (
      <Centered>
        <Text style={styles.title}>No shop yet</Text>
        <Text style={[styles.muted, { textAlign: 'center', marginTop: spacing.sm }]}>
          Open one on the web and it will show up here.
        </Text>
        <View style={{ marginTop: spacing.lg }}>
          <Button label="Sign out" variant="secondary" onPress={onSignOut} />
        </View>
      </Centered>
    )
  }

  if (adding) {
    return (
      <AddProduct
        storeId={store.id}
        currency={store.currency}
        onDone={() => {
          setAdding(false)
          reload()
        }}
        onCancel={() => setAdding(false)}
      />
    )
  }

  const drafts = data?.store?.drafts ?? []
  const published = data?.store?.published ?? []
  const products = [...drafts, ...published]

  return (
    <View style={styles.page}>
      <View style={styles.header}>
        <View style={{ flex: 1 }}>
          <Text style={styles.title}>{store.name}</Text>
          <Text style={styles.muted}>
            {published.length} published · {drafts.length} draft
            {drafts.length === 1 ? '' : 's'}
          </Text>
        </View>
        <Pressable onPress={onSignOut} hitSlop={12}>
          <Text style={styles.signOut}>Sign out</Text>
        </Pressable>
      </View>

      <FlatList
        data={products}
        keyExtractor={(item) => item.id}
        contentContainerStyle={{ padding: spacing.lg, paddingTop: 0 }}
        // Pull to refresh, because that is what a phone user reaches for
        // before they look for a button.
        refreshControl={
          <RefreshControl refreshing={fetching} onRefresh={reload} tintColor={colors.muted} />
        }
        ListEmptyComponent={
          <Text style={[styles.muted, { marginTop: spacing.xl, textAlign: 'center' }]}>
            Nothing here yet. Photograph something to start.
          </Text>
        }
        renderItem={({ item }) => (
          <ProductRow
            product={item}
            currency={store.currency}
            busy={updating && changing === item.id}
            disabled={updating}
            onTogglePublished={() => togglePublished(item)}
          />
        )}
      />

      <View style={styles.footer}>
        <Button label="Photograph a product" onPress={() => setAdding(true)} />
      </View>
    </View>
  )
}

function ProductRow({
  product,
  currency,
  busy,
  disabled,
  onTogglePublished,
}: {
  product: ProductCardFragment
  currency: string
  busy: boolean
  disabled: boolean
  onTogglePublished: () => void
}) {
  const image = product.images[0]
  const live = product.status === ProductStatus.Active

  return (
    <View style={styles.row}>
      {image ? (
        <Image source={{ uri: image.url }} style={styles.thumb} accessibilityLabel={image.altText} />
      ) : (
        <View style={[styles.thumb, styles.thumbEmpty]} />
      )}

      <View style={{ flex: 1, marginHorizontal: spacing.md }}>
        <Text style={styles.rowTitle} numberOfLines={1}>
          {product.name}
        </Text>
        <Text style={styles.muted}>{formatMoney(product.priceCents, currency)}</Text>
        <Text style={[styles.status, live && styles.statusLive]}>
          {live ? 'On the shop' : 'Draft'}
        </Text>
      </View>

      <Pressable
        onPress={onTogglePublished}
        disabled={disabled}
        hitSlop={8}
        style={({ pressed }) => [
          styles.action,
          live && styles.actionQuiet,
          (pressed || disabled) && { opacity: 0.6 },
        ]}
        accessibilityRole="button"
        accessibilityLabel={live ? `Unpublish ${product.name}` : `Publish ${product.name}`}
        accessibilityState={{ busy, disabled }}
      >
        {busy ? (
          <ActivityIndicator size="small" color={live ? colors.muted : '#fff'} />
        ) : (
          <Text style={[styles.actionText, live && styles.actionTextQuiet]}>
            {live ? 'Unpublish' : 'Publish'}
          </Text>
        )}
      </Pressable>
    </View>
  )
}

const styles = StyleSheet.create({
  page: { flex: 1, backgroundColor: colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    padding: spacing.lg,
    paddingBottom: spacing.md,
  },
  title: { fontSize: 28, color: colors.ink, fontWeight: '400' },
  muted: { color: colors.muted, fontSize: 14 },
  signOut: { color: colors.muted, fontSize: 14, paddingTop: 6 },

  row: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: colors.raised,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.line,
    padding: spacing.sm,
    marginBottom: spacing.sm,
  },
  thumb: { width: 60, height: 60, borderRadius: radius.sm, backgroundColor: colors.accentSoft },
  thumbEmpty: { borderWidth: 1, borderColor: colors.line },
  rowTitle: { fontSize: 16, color: colors.ink, marginBottom: 2 },

  status: { fontSize: 12, color: colors.muted, marginTop: 3 },
  statusLive: { color: colors.accent },

  action: {
    minWidth: 88,
    paddingVertical: 9,
    paddingHorizontal: 12,
    borderRadius: radius.sm,
    backgroundColor: colors.accent,
    alignItems: 'center',
    justifyContent: 'center',
  },
  actionQuiet: { backgroundColor: colors.raised, borderWidth: 1, borderColor: colors.line },
  actionText: { color: '#fff', fontSize: 13, fontWeight: '600' },
  actionTextQuiet: { color: colors.muted },

  footer: {
    padding: spacing.lg,
    borderTopWidth: 1,
    borderTopColor: colors.line,
    backgroundColor: colors.paper,
  },
})
