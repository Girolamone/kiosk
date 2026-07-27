import { useState } from 'react'
import {
  Image,
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from 'react-native'
import * as ImagePicker from 'expo-image-picker'
import { useMutation } from 'urql'
import {
  CreateProductDocument,
  GenerateProductCopyDocument,
  uploadImage,
  type UploadedImage,
} from '@kiosk/shared'
import { apiOptions } from '../api'
import { Button, Field, Notice } from '../components'
import { colors, radius, spacing } from '../theme'

/**
 * The screen the whole app exists for: photograph the thing, and the listing
 * writes itself. On the web this means finding a file; here it means holding
 * up a phone.
 */
export function AddProduct({
  storeId,
  currency,
  onDone,
  onCancel,
}: {
  storeId: string
  currency: string
  onDone: () => void
  onCancel: () => void
}) {
  const [photo, setPhoto] = useState<{ uri: string; uploaded: UploadedImage } | null>(null)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [altText, setAltText] = useState('')
  const [price, setPrice] = useState('')

  const [stage, setStage] = useState<'idle' | 'uploading' | 'reading'>('idle')
  const [problem, setProblem] = useState<string | null>(null)
  const [note, setNote] = useState<string | null>(null)

  const [, generateCopy] = useMutation(GenerateProductCopyDocument)
  const [{ fetching: saving, error: saveError }, createProduct] = useMutation(CreateProductDocument)

  async function pick(source: 'camera' | 'library') {
    setProblem(null)
    setNote(null)

    const permission =
      source === 'camera'
        ? await ImagePicker.requestCameraPermissionsAsync()
        : await ImagePicker.requestMediaLibraryPermissionsAsync()

    if (!permission.granted) {
      setProblem(
        source === 'camera'
          ? 'Kiosk needs camera access to photograph a product. You can grant it in Settings.'
          : 'Kiosk needs access to your photos. You can grant it in Settings.',
      )
      return
    }

    const options: ImagePicker.ImagePickerOptions = {
      mediaTypes: ['images'],
      // Phone cameras produce files far larger than the 8 MB the API
      // accepts, and a listing photo does not need them.
      quality: 0.7,
      allowsEditing: true,
      aspect: [4, 5],
    }

    const result =
      source === 'camera'
        ? await ImagePicker.launchCameraAsync(options)
        : await ImagePicker.launchImageLibraryAsync(options)

    if (result.canceled) return
    const asset = result.assets[0]

    setStage('uploading')
    try {
      const uploaded = await uploadImage(apiOptions, {
        uri: asset.uri,
        name: asset.fileName ?? 'photo.jpg',
        type: asset.mimeType ?? 'image/jpeg',
      })
      setPhoto({ uri: asset.uri, uploaded })

      setStage('reading')
      const written = await generateCopy({ imageKey: uploaded.key })
      const copy = written.data?.generateProductCopy
      if (copy) {
        // Only fill what has not been typed already.
        setName((current) => current || copy.title)
        setDescription((current) => current || copy.description)
        setAltText((current) => current || copy.altText)
      } else {
        // The photo is saved either way. Say so and leave the form usable.
        setNote('Could not write the listing this time — the photo is saved, so fill it in yourself.')
      }
    } catch (err) {
      setProblem(err instanceof Error ? err.message : 'That photo could not be uploaded.')
    } finally {
      setStage('idle')
    }
  }

  async function save() {
    const result = await createProduct({
      input: {
        storeId,
        name,
        description,
        priceCents: Math.round(Number(price) * 100),
        // The key, not the URL: the server builds the URL from it.
        image: photo ? { key: photo.uploaded.key, altText } : null,
      },
    })
    if (!result.error) onDone()
  }

  const busy = stage !== 'idle'

  return (
    <KeyboardAvoidingView
      style={{ flex: 1 }}
      behavior={Platform.OS === 'ios' ? 'padding' : undefined}
    >
      <ScrollView
        style={styles.page}
        contentContainerStyle={{ padding: spacing.lg }}
        keyboardShouldPersistTaps="handled"
      >
        <View style={styles.header}>
          <Text style={styles.title}>New product</Text>
          <Pressable onPress={onCancel} hitSlop={12}>
            <Text style={styles.cancel}>Cancel</Text>
          </Pressable>
        </View>

        <Pressable
          style={styles.photo}
          onPress={() => pick('camera')}
          disabled={busy}
          accessibilityRole="button"
          accessibilityLabel="Take a photograph of the product"
        >
          {photo ? (
            <Image source={{ uri: photo.uri }} style={styles.photoImage} />
          ) : (
            <Text style={styles.photoHint}>Tap to photograph</Text>
          )}
        </Pressable>

        <View style={styles.photoActions}>
          <View style={{ flex: 1 }}>
            <Button
              label={photo ? 'Retake' : 'Camera'}
              variant="secondary"
              onPress={() => pick('camera')}
              disabled={busy}
            />
          </View>
          <View style={{ width: spacing.sm }} />
          <View style={{ flex: 1 }}>
            <Button
              label="Choose photo"
              variant="secondary"
              onPress={() => pick('library')}
              disabled={busy}
            />
          </View>
        </View>

        {stage !== 'idle' && (
          <Notice>{stage === 'uploading' ? 'Uploading the photo…' : 'Reading the photo…'}</Notice>
        )}
        {note && <Notice>{note}</Notice>}
        <Notice tone="error">
          {problem ?? saveError?.message.replace(/^\[GraphQL\]\s*/, '') ?? null}
        </Notice>

        <Field label="Name" value={name} onChangeText={setName} placeholder="What is it?" />
        <Field
          label="Description"
          value={description}
          onChangeText={setDescription}
          multiline
          numberOfLines={4}
          style={styles.multiline}
        />
        {photo && (
          <Field
            label="Photo description"
            hint="Read aloud by screen readers. Describe the photo, not the sales pitch."
            value={altText}
            onChangeText={setAltText}
          />
        )}
        <Field
          label={`Price (${currency})`}
          value={price}
          onChangeText={setPrice}
          keyboardType="decimal-pad"
          placeholder="0.00"
        />

        <Button
          label="Save as draft"
          onPress={save}
          busy={saving}
          disabled={busy || !name || !price}
        />
      </ScrollView>
    </KeyboardAvoidingView>
  )
}

const styles = StyleSheet.create({
  page: { flex: 1, backgroundColor: colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: spacing.lg,
  },
  title: { fontSize: 26, color: colors.ink },
  cancel: { color: colors.muted, fontSize: 15 },

  photo: {
    aspectRatio: 4 / 5,
    borderRadius: radius.md,
    borderWidth: 1,
    borderColor: colors.line,
    borderStyle: 'dashed',
    backgroundColor: colors.raised,
    alignItems: 'center',
    justifyContent: 'center',
    overflow: 'hidden',
  },
  photoImage: { width: '100%', height: '100%' },
  photoHint: { color: colors.muted, fontSize: 15 },
  photoActions: { flexDirection: 'row', marginTop: spacing.sm, marginBottom: spacing.md },

  multiline: { height: 100, textAlignVertical: 'top', paddingTop: 10 },
})
