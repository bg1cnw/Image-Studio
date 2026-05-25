package top.gptcodex.imagestudio.android

import android.annotation.SuppressLint
import android.net.Uri
import android.os.Bundle
import android.util.Log
import android.webkit.ConsoleMessage
import android.webkit.WebChromeClient
import android.webkit.WebResourceRequest
import android.webkit.WebResourceResponse
import android.webkit.WebSettings
import android.webkit.WebView
import androidx.activity.result.contract.ActivityResultContracts
import androidx.appcompat.app.AppCompatActivity
import androidx.webkit.WebViewAssetLoader
import androidx.webkit.WebViewClientCompat

class MainActivity : AppCompatActivity() {
    private lateinit var webView: WebView
    private lateinit var bridge: AndroidImageStudioBridge
    private lateinit var assetLoader: WebViewAssetLoader
    private val openImageLauncher = registerForActivityResult(ActivityResultContracts.OpenDocument()) { uri: Uri? ->
        if (::bridge.isInitialized) bridge.onOpenImageDialogResult(uri)
    }
    private val importHistoryLauncher = registerForActivityResult(ActivityResultContracts.OpenDocument()) { uri: Uri? ->
        if (::bridge.isInitialized) bridge.onImportHistoryResult(uri)
    }

    @SuppressLint("SetJavaScriptEnabled")
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)

        webView = findViewById(R.id.webview)
        bridge = AndroidImageStudioBridge(
            this,
            webView,
            launchOpenImageDialog = {
                openImageLauncher.launch(arrayOf("image/*"))
            },
            launchImportHistory = {
                importHistoryLauncher.launch(arrayOf("application/json", "text/plain", "*/*"))
            },
        )
        assetLoader = WebViewAssetLoader.Builder()
            .addPathHandler("/assets/", WebViewAssetLoader.AssetsPathHandler(this))
            .build()

        webView.settings.apply {
            javaScriptEnabled = true
            domStorageEnabled = true
            allowFileAccess = true
            allowContentAccess = true
            mixedContentMode = WebSettings.MIXED_CONTENT_ALWAYS_ALLOW
            useWideViewPort = false
            loadWithOverviewMode = false
            mediaPlaybackRequiresUserGesture = false
        }
        WebView.setWebContentsDebuggingEnabled(true)
        webView.webChromeClient = object : WebChromeClient() {
            override fun onConsoleMessage(consoleMessage: ConsoleMessage): Boolean {
                Log.d(
                    "ImageStudioWebView",
                    "${consoleMessage.messageLevel()}: ${consoleMessage.message()} @ ${consoleMessage.sourceId()}:${consoleMessage.lineNumber()}",
                )
                return super.onConsoleMessage(consoleMessage)
            }
        }
        webView.webViewClient = object : WebViewClientCompat() {
            override fun shouldInterceptRequest(
                view: WebView,
                request: WebResourceRequest,
            ): WebResourceResponse? = assetLoader.shouldInterceptRequest(request.url)
        }
        webView.addJavascriptInterface(bridge, "AndroidImageStudio")
        webView.loadUrl("https://appassets.androidplatform.net/assets/index.html?target=${BuildConfig.TARGET_PLATFORM}")
    }

    override fun onDestroy() {
        webView.removeJavascriptInterface("AndroidImageStudio")
        webView.destroy()
        super.onDestroy()
    }
}
