package top.gptcodex.imagestudio.android

import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.Environment
import android.util.Base64
import android.webkit.JavascriptInterface
import android.webkit.WebView
import androidx.core.content.FileProvider
import org.json.JSONObject
import org.json.JSONArray
import android.provider.OpenableColumns
import java.io.File
import java.io.FileInputStream
import java.io.FileOutputStream
import java.io.InputStream
import java.net.HttpURLConnection
import java.net.URL
import java.text.SimpleDateFormat
import java.util.Date
import java.util.concurrent.ConcurrentHashMap
import java.util.Locale
import kotlin.concurrent.thread

class AndroidImageStudioBridge(
    private val context: Context,
    private val webView: WebView,
    private val launchOpenImageDialog: () -> Unit,
    private val launchImportHistory: () -> Unit,
) {
    private val prefs = context.getSharedPreferences("image_studio_android", Context.MODE_PRIVATE)
    private val outputDirKey = "output_dir"
    private var pendingOpenImageRequestId: String? = null
    private var pendingImportHistoryRequestId: String? = null
    private val httpRequests = ConcurrentHashMap<String, HttpURLConnection>()

    companion object {
        private const val maxDialogReadBytes: Long = 50L * 1024L * 1024L
    }

    @JavascriptInterface
    fun invoke(requestId: String, method: String, payloadJson: String) {
        try {
            val args = JSONArray(payloadJson)
            when (method) {
                "OpenImageDialog" -> {
                    if (pendingOpenImageRequestId != null) {
                        throw IllegalStateException("图片选择已在进行中")
                    }
                    pendingOpenImageRequestId = requestId
                    launchOpenImageDialog()
                    return
                }
                "ImportHistoryFromFile" -> {
                    if (pendingImportHistoryRequestId != null) {
                        throw IllegalStateException("历史导入已在进行中")
                    }
                    pendingImportHistoryRequestId = requestId
                    launchImportHistory()
                    return
                }
            }
            val result: Any? = when (method) {
                "GetOutputDir" -> getOutputDir()
                "SetOutputDir" -> {
                    setOutputDir(args.optString(0, ""))
                    null
                }
                "ChooseOutputDir" -> getOutputDir()
                "GetStoredAPIKey" -> getStoredApiKey(args.optString(0))
                "SetStoredAPIKey" -> {
                    setStoredApiKey(args.optString(0), args.optString(1))
                    null
                }
                "DeleteStoredAPIKey" -> {
                    deleteStoredApiKey(args.optString(0))
                    null
                }
                "OpenExternalURL" -> {
                    openExternalUrl(args.optString(0))
                    null
                }
                "OpenOutputDir" -> {
                    openOutputDir()
                    null
                }
                "ImportImageFromB64" -> importImageFromB64(args.optString(0), args.optString(1))
                "ReadImageAsBase64" -> readImageAsBase64(args.optString(0))
                "ReadTextFile" -> readTextFile(args.optString(0))
                "OpenFile" -> {
                    openFile(args.optString(0))
                    null
                }
                "ExportHistoryToFile" -> exportHistory(args.optString(0))
                "SaveImageAs" -> saveImage(args.optString(0), args.optString(1))
                "HttpRequestText" -> {
                    val payload = args.optJSONObject(0) ?: throw IllegalArgumentException("缺少 HTTP 请求参数")
                    runHttpRequestText(requestId, payload)
                }
                "CancelHttpRequest" -> {
                    cancelHttpRequest(args.optString(0))
                    null
                }
                else -> throw UnsupportedOperationException("$method is not implemented in Android shell yet")
            }
            resolve(requestId, result)
        } catch (_: EarlyResolve) {
            return
        } catch (error: Exception) {
            reject(requestId, error.message ?: error.javaClass.simpleName)
        }
    }

    @JavascriptInterface
    fun getOutputDir(): String {
        return prefs.getString(outputDirKey, defaultOutputDir().absolutePath) ?: defaultOutputDir().absolutePath
    }

    @JavascriptInterface
    fun setOutputDir(path: String) {
        val dir = if (path.isBlank()) defaultOutputDir() else File(path)
        dir.mkdirs()
        prefs.edit().putString(outputDirKey, dir.absolutePath).apply()
    }

    @JavascriptInterface
    fun importImageFromB64(imageB64: String, suggestedName: String): Map<String, Any> {
        val bytes = Base64.decode(imageB64, Base64.DEFAULT)
        val file = writeImportedBytes(bytes, suggestedName)
        return mapOf(
            "path" to file.absolutePath,
            "imageB64" to imageB64,
        )
    }

    @JavascriptInterface
    fun getStoredApiKey(user: String): String {
        return prefs.getString("apikey_$user", "") ?: ""
    }

    @JavascriptInterface
    fun setStoredApiKey(user: String, value: String) {
        if (value.isBlank()) prefs.edit().remove("apikey_$user").apply()
        else prefs.edit().putString("apikey_$user", value.trim()).apply()
    }

    @JavascriptInterface
    fun deleteStoredApiKey(user: String) {
        prefs.edit().remove("apikey_$user").apply()
    }

    @JavascriptInterface
    fun openOutputDir(): String {
        val dir = File(getOutputDir()).apply { mkdirs() }
        val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", dir)
        val intent = Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(uri, "*/*")
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }
        context.startActivity(intent)
        return dir.absolutePath
    }

    @JavascriptInterface
    fun openExternalUrl(url: String) {
        val intent = Intent(Intent.ACTION_VIEW, Uri.parse(url)).apply {
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK)
        }
        context.startActivity(intent)
    }

    @JavascriptInterface
    fun exportHistory(jsonContent: String): String {
        val file = File(getOutputDir(), "image-studio-history-${timestamp()}.json")
        file.parentFile?.mkdirs()
        file.writeText(jsonContent)
        return file.absolutePath
    }

    @JavascriptInterface
    fun readImageAsBase64(path: String): String {
        val bytes = openInputStreamForPath(path).use { it.readBytes() }
        return Base64.encodeToString(bytes, Base64.NO_WRAP)
    }

    @JavascriptInterface
    fun readTextFile(path: String): String {
        return openInputStreamForPath(path).bufferedReader().use { it.readText() }
    }

    @JavascriptInterface
    fun openFile(path: String) {
        val uriAndMime = uriAndMimeForPath(path)
        val intent = Intent(Intent.ACTION_VIEW).apply {
            setDataAndType(uriAndMime.first, uriAndMime.second)
            addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }
        context.startActivity(intent)
    }

    @JavascriptInterface
    fun saveImage(imageB64: String, suggestedName: String): String {
        val name = if (suggestedName.endsWith(".png", true)) suggestedName else "$suggestedName.png"
        val file = File(getOutputDir(), name)
        file.parentFile?.mkdirs()
        file.writeBytes(Base64.decode(imageB64, Base64.DEFAULT))
        return file.absolutePath
    }

    private fun runHttpRequestText(requestId: String, payload: JSONObject): Nothing {
        val requestKey = payload.optString("requestKey").ifBlank { requestId }
        val url = payload.optString("url").trim()
        val method = payload.optString("method", "GET").trim().uppercase(Locale.US)
        val headersJson = payload.optJSONObject("headers")
        val bodyBase64 = payload.optString("bodyBase64")
        val contentType = payload.optString("contentType")
        thread(name = "image-studio-http-$requestKey") {
            try {
                val connection = (URL(url).openConnection() as HttpURLConnection).apply {
                    requestMethod = method
                    instanceFollowRedirects = true
                    connectTimeout = 30_000
                    readTimeout = 180_000
                    doInput = true
                }
                httpRequests[requestKey] = connection
                if (headersJson != null) {
                    val keys = headersJson.keys()
                    while (keys.hasNext()) {
                        val key = keys.next()
                        connection.setRequestProperty(key, headersJson.optString(key))
                    }
                }
                if (contentType.isNotBlank() && connection.getRequestProperty("Content-Type").isNullOrBlank()) {
                    connection.setRequestProperty("Content-Type", contentType)
                }
                val requestBytes = if (bodyBase64.isBlank()) ByteArray(0) else Base64.decode(bodyBase64, Base64.DEFAULT)
                if (requestBytes.isNotEmpty()) {
                    connection.doOutput = true
                    connection.outputStream.use { it.write(requestBytes) }
                }
                val status = connection.responseCode
                val stream = if (status >= 400) connection.errorStream else connection.inputStream
                val body = stream?.bufferedReader()?.use { it.readText() } ?: ""
                val result = mapOf(
                    "status" to status,
                    "body" to body,
                    "contentType" to (connection.contentType ?: ""),
                )
                resolve(requestId, result)
            } catch (error: Exception) {
                reject(requestId, error.message ?: error.javaClass.simpleName)
            } finally {
                httpRequests.remove(requestKey)?.disconnect()
            }
        }
        throw EarlyResolve()
    }

    private fun cancelHttpRequest(requestKey: String) {
        httpRequests.remove(requestKey)?.disconnect()
    }

    fun onOpenImageDialogResult(uri: Uri?) {
        val requestId = pendingOpenImageRequestId ?: return
        pendingOpenImageRequestId = null
        if (uri == null) {
            resolve(requestId, mapOf("path" to "", "size" to 0, "imageB64" to ""))
            return
        }
        try {
            val suggestedName = queryDisplayName(uri) ?: "import-${timestamp()}.png"
            val copied = copyUriToImports(uri, suggestedName)
            resolve(
                requestId,
                mapOf(
                    "path" to copied.file.absolutePath,
                    "size" to copied.size,
                    "imageB64" to copied.imageB64,
                ),
            )
        } catch (error: Exception) {
            reject(requestId, error.message ?: error.javaClass.simpleName)
        }
    }

    fun onImportHistoryResult(uri: Uri?) {
        val requestId = pendingImportHistoryRequestId ?: return
        pendingImportHistoryRequestId = null
        if (uri == null) {
            resolve(requestId, "")
            return
        }
        try {
            val text = context.contentResolver.openInputStream(uri)?.bufferedReader()?.use { it.readText() } ?: ""
            resolve(requestId, text)
        } catch (error: Exception) {
            reject(requestId, error.message ?: error.javaClass.simpleName)
        }
    }

    private fun defaultOutputDir(): File {
        val pictures = context.getExternalFilesDir(Environment.DIRECTORY_PICTURES)
        return File(pictures ?: context.filesDir, "ImageStudio")
    }

    private fun importsDir(): File {
        return File(context.filesDir, "imports").apply { mkdirs() }
    }

    private fun sanitizeFileName(name: String, fallback: String): String {
        val trimmed = name.trim()
        if (trimmed.isEmpty()) return fallback
        return trimmed.replace(Regex("[^A-Za-z0-9._\\-\\u4E00-\\u9FFF]+"), "-")
    }

    private fun queryDisplayName(uri: Uri): String? {
        context.contentResolver.query(uri, arrayOf(OpenableColumns.DISPLAY_NAME), null, null, null)?.use { cursor ->
            val index = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
            if (index >= 0 && cursor.moveToFirst()) {
                val name = cursor.getString(index)
                if (!name.isNullOrBlank()) return name
            }
        }
        return null
    }

    private data class CopiedImport(
        val file: File,
        val size: Long,
        val imageB64: String,
    )

    private fun copyUriToImports(uri: Uri, suggestedName: String): CopiedImport {
        val name = sanitizeFileName(suggestedName, "import-${timestamp()}.png")
        val target = File(importsDir(), "${timestamp()}-$name")
        var total = 0L
        val preview = java.io.ByteArrayOutputStream()
        context.contentResolver.openInputStream(uri)?.use { input ->
            FileOutputStream(target).use { output ->
                val buffer = ByteArray(8192)
                while (true) {
                    val read = input.read(buffer)
                    if (read <= 0) break
                    output.write(buffer, 0, read)
                    total += read
                    if (total <= maxDialogReadBytes) {
                        preview.write(buffer, 0, read)
                    }
                }
            }
        } ?: throw IllegalStateException("无法读取所选文件")
        val imageB64 = if (total in 1..maxDialogReadBytes) {
            Base64.encodeToString(preview.toByteArray(), Base64.NO_WRAP)
        } else {
            ""
        }
        return CopiedImport(target, total, imageB64)
    }

    private fun writeImportedBytes(bytes: ByteArray, suggestedName: String): File {
        val safeName = sanitizeFileName(suggestedName, "import-${timestamp()}.png")
        val file = File(importsDir(), "${timestamp()}-$safeName")
        file.writeBytes(bytes)
        return file
    }

    private fun openInputStreamForPath(path: String): InputStream {
        val trimmed = path.trim()
        if (trimmed.startsWith("content://")) {
            return context.contentResolver.openInputStream(Uri.parse(trimmed))
                ?: throw IllegalArgumentException("无法读取内容 URI: $trimmed")
        }
        return FileInputStream(File(trimmed))
    }

    private fun uriAndMimeForPath(path: String): Pair<Uri, String> {
        val trimmed = path.trim()
        val mime = when {
            trimmed.endsWith(".png", true) -> "image/png"
            trimmed.endsWith(".jpg", true) || trimmed.endsWith(".jpeg", true) -> "image/jpeg"
            trimmed.endsWith(".webp", true) -> "image/webp"
            trimmed.endsWith(".json", true) -> "application/json"
            trimmed.endsWith(".txt", true) -> "text/plain"
            else -> "*/*"
        }
        if (trimmed.startsWith("content://")) {
            return Uri.parse(trimmed) to mime
        }
        val file = File(trimmed)
        val uri = FileProvider.getUriForFile(context, "${context.packageName}.fileprovider", file)
        return uri to mime
    }

    private fun timestamp(): String = SimpleDateFormat("yyyyMMdd-HHmmss", Locale.US).format(Date())

    private fun resolve(requestId: String, payload: Any?) {
        val serialized = when (payload) {
            null -> "null"
            is String -> JSONObject.quote(payload)
            is Number, is Boolean -> payload.toString()
            else -> JSONObject.wrap(payload)?.toString() ?: "null"
        }
        webView.post {
            webView.evaluateJavascript("window.__imageStudioNativeResolve(${JSONObject.quote(requestId)}, $serialized)", null)
        }
    }

    private fun reject(requestId: String, message: String) {
        webView.post {
            webView.evaluateJavascript(
                "window.__imageStudioNativeReject(${JSONObject.quote(requestId)}, ${JSONObject.quote(message)})",
                null,
            )
        }
    }

    private class EarlyResolve : RuntimeException()
}
