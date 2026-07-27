import { useState } from 'react'
import { FlatList, Image, Pressable, RefreshControl, StyleSheet, Text, View } from 'react-native'
import { useQuery } from 'urql'
import {
  MyStoresDocument,
  StoreProductsDocument,
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

  const reload = () => refetch({ requestPolicy: 'network-only' })

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
        renderItem={({ item }) => <ProductRow product={item} currency={store.currency} />}
      />

      <View style={styles.footer}>
        <Button label="Photograph a product" onPress={() => setAdding(true)} />
      </View>
    </View>
  )
}

function ProductRow({ product, currency }: { product: ProductCardFragment; currency: string }) {
  const image = product.images[0]
  return (
    <View style={styles.row}>
      {image ? (
        <Image source={{ uri: image.url }} style={styles.thumb} accessibilityLabel={image.altText} />
      ) : (
        <View style={[styles.thumb, styles.thumbEmpty]} />
      )}

      <View style={{ flex: 1, marginLeft: spacing.md }}>
        <Text style={styles.rowTitle} numberOfLines={1}>
          {product.name}
        </Text>
        <Text style={styles.muted}>{formatMoney(product.priceCents, currency)}</Text>
      </View>

      {product.status === 'DRAFT' && (
        <View style={styles.badge}>
          <Text style={styles.badgeText}>Draft</Text>
        </View>
      )}
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

  badge: {
    backgroundColor: colors.accentSoft,
    paddingHorizontal: 8,
    paddingVertical: 3,
    borderRadius: 999,
  },
  badgeText: { color: colors.accent, fontSize: 11, fontWeight: '600' },

  footer: {
    padding: spacing.lg,
    borderTopWidth: 1,
    borderTopColor: colors.line,
    backgroundColor: colors.paper,
  },
})
