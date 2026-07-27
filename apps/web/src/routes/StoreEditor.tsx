import { useState } from 'react'
import { Link, Navigate, useParams } from 'react-router'
import { useMutation, useQuery } from 'urql'
import {
  CreateProductDocument,
  GenerateProductCopyDocument,
  ProductStatus,
  StoreProductsDocument,
  UpdateProductDocument,
  formatMoney,
  uploadImage,
  type ProductCardFragment,
  type UploadedImage,
} from '@kiosk/shared'
import { Button, Card, Empty, ErrorNote, Field, Input, Spinner, Textarea } from '../components/ui'
import { readableError, useSession } from '../lib/session'

export function StoreEditor() {
  const { slug = '' } = useParams()
  const { user, loading } = useSession()
  const [{ data, fetching }, refetch] = useQuery({
    query: StoreProductsDocument,
    variables: { slug },
  })

  if (loading) {
    return (
      <p className="flex items-center gap-2 py-16 text-muted">
        <Spinner /> Checking your session…
      </p>
    )
  }
  if (!user) return <Navigate to="/signin" replace />

  const store = data?.store
  if (fetching && !data) {
    return (
      <p className="flex items-center gap-2 py-16 text-muted">
        <Spinner /> Loading…
      </p>
    )
  }
  if (!store) return <Empty title="No such shop">Or it is not yours.</Empty>

  const reload = () => refetch({ requestPolicy: 'network-only' })

  return (
    <div>
      <div className="flex flex-wrap items-baseline justify-between gap-3">
        <h1 className="text-3xl">{store.name}</h1>
        <Link to={`/s/${store.slug}`} className="text-sm text-muted underline">
          View the public shop
        </Link>
      </div>

      <NewProductForm storeId={store.id} currency={store.currency} onCreated={reload} />

      <ProductList
        title="Drafts"
        note="Only you can see these."
        products={store.drafts}
        currency={store.currency}
        onChanged={reload}
      />
      <ProductList
        title="Published"
        note="Live on your shop page."
        products={store.published}
        currency={store.currency}
        onChanged={reload}
      />
    </div>
  )
}

function NewProductForm({
  storeId,
  currency,
  onCreated,
}: {
  storeId: string
  currency: string
  onCreated: () => void
}) {
  const [image, setImage] = useState<UploadedImage | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [altText, setAltText] = useState('')
  const [price, setPrice] = useState('')

  const [uploading, setUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [copyNote, setCopyNote] = useState<string | null>(null)

  const [generateState, generateCopy] = useMutation(GenerateProductCopyDocument)
  const [createState, createProduct] = useMutation(CreateProductDocument)

  async function pickImage(file: File) {
    setUploadError(null)
    setCopyNote(null)
    setUploading(true)
    try {
      const uploaded = await uploadImage({ baseUrl: '' }, file)
      setImage(uploaded)
      await writeCopyFor(uploaded)
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  async function writeCopyFor(uploaded: UploadedImage) {
    const result = await generateCopy({ imageKey: uploaded.key })
    const copy = result.data?.generateProductCopy
    if (!copy) {
      // Generation is an accelerator, not a requirement. Say so and leave the
      // form usable rather than treating this as a failed upload.
      setCopyNote('Could not write the copy this time. The photo is saved, so fill the rest in yourself.')
      return
    }
    // Only fill fields the seller has not already written in.
    setName((current) => current || copy.title)
    setDescription((current) => current || copy.description)
    setAltText((current) => current || copy.altText)
    setCopyNote(null)
  }

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    const priceCents = Math.round(Number(price) * 100)
    const result = await createProduct({
      input: {
        storeId,
        name,
        description,
        priceCents,
        // The key, not the URL: the server builds the URL from it.
        image: image ? { key: image.key, altText } : null,
      },
    })
    if (result.error) return

    setImage(null)
    setName('')
    setDescription('')
    setAltText('')
    setPrice('')
    setCopyNote(null)
    onCreated()
  }

  const generating = generateState.fetching

  return (
    <section className="mt-8 border-y border-line py-8">
      <h2 className="text-2xl">Add a product</h2>
      <p className="mt-1 text-sm text-muted">
        Start with a photo and the listing writes itself. Every field stays editable.
      </p>

      <form onSubmit={submit} className="mt-5 grid gap-6 sm:grid-cols-[200px_1fr]">
        <div>
          <div className="aspect-square overflow-hidden rounded-lg border border-dashed border-line bg-white">
            {image ? (
              <img src={image.url} alt="" className="size-full object-cover" />
            ) : (
              <div className="flex size-full items-center justify-center px-3 text-center text-xs text-muted">
                No photo yet
              </div>
            )}
          </div>

          <label className="mt-3 block">
            <span className="sr-only">Product photo</span>
            <input
              type="file"
              accept="image/jpeg,image/png,image/webp,image/gif"
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) void pickImage(file)
              }}
              className="block w-full text-xs text-muted file:mr-2 file:rounded-md file:border file:border-line file:bg-white file:px-3 file:py-1.5 file:text-xs file:text-ink"
            />
          </label>

          {(uploading || generating) && (
            <p className="mt-2 flex items-center gap-2 text-xs text-muted">
              <Spinner />
              {uploading ? 'Uploading…' : 'Reading the photo…'}
            </p>
          )}
        </div>

        <div className="space-y-4">
          <Field label="Name">
            <Input value={name} onChange={(e) => setName(e.target.value)} required />
          </Field>

          <Field label="Description">
            <Textarea
              rows={4}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </Field>

          {image && (
            <Field
              label="Photo description"
              hint="Read aloud by screen readers and shown if the image fails to load. Describe the photo, not the sales pitch."
            >
              <Input value={altText} onChange={(e) => setAltText(e.target.value)} />
            </Field>
          )}

          <Field label={`Price (${currency})`}>
            <Input
              type="number"
              min="0"
              step="0.01"
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              required
            />
          </Field>

          <ErrorNote>{uploadError ?? readableError(createState.error)}</ErrorNote>
          {copyNote && <p className="text-sm text-muted">{copyNote}</p>}

          <Button type="submit" busy={createState.fetching}>
            Save as draft
          </Button>
        </div>
      </form>
    </section>
  )
}

function ProductList({
  title,
  note,
  products,
  currency,
  onChanged,
}: {
  title: string
  note: string
  products: ProductCardFragment[]
  currency: string
  onChanged: () => void
}) {
  const [{ fetching }, updateProduct] = useMutation(UpdateProductDocument)

  async function setStatus(id: string, status: ProductStatus) {
    const result = await updateProduct({ input: { id, status } })
    if (!result.error) onChanged()
  }

  return (
    <section className="mt-10">
      <div className="flex items-baseline gap-3">
        <h2 className="text-2xl">{title}</h2>
        <span className="text-sm text-muted">{note}</span>
      </div>

      {products.length === 0 ? (
        <p className="mt-3 text-sm text-muted">Nothing here.</p>
      ) : (
        <ul className="mt-4 space-y-3">
          {products.map((product) => (
            <li key={product.id}>
              <Card className="flex items-center gap-4 p-3">
                <div className="size-16 shrink-0 overflow-hidden rounded border border-line">
                  {product.images[0] ? (
                    <img
                      src={product.images[0].url}
                      alt={product.images[0].altText || product.name}
                      className="size-full object-cover"
                    />
                  ) : (
                    <div className="size-full bg-accent-soft" />
                  )}
                </div>

                <div className="min-w-0 flex-1">
                  <p className="truncate font-medium">{product.name}</p>
                  <p className="text-sm tabular-nums text-muted">
                    {formatMoney(product.priceCents, currency)}
                  </p>
                </div>

                <Button
                  variant="secondary"
                  busy={fetching}
                  onClick={() =>
                    setStatus(
                      product.id,
                      product.status === ProductStatus.Active
                        ? ProductStatus.Draft
                        : ProductStatus.Active,
                    )
                  }
                >
                  {product.status === ProductStatus.Active ? 'Unpublish' : 'Publish'}
                </Button>
              </Card>
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
