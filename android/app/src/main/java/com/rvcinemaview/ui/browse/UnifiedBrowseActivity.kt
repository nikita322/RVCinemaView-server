package com.rvcinemaview.ui.browse

import android.content.Intent
import android.content.res.Configuration
import android.graphics.Typeface
import android.os.Bundle
import android.view.View
import android.widget.HorizontalScrollView
import android.widget.ImageButton
import android.widget.LinearLayout
import android.widget.ProgressBar
import android.widget.TextView
import androidx.activity.OnBackPressedCallback
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import androidx.lifecycle.lifecycleScope
import androidx.recyclerview.widget.GridLayoutManager
import androidx.recyclerview.widget.RecyclerView
import com.facebook.shimmer.ShimmerFrameLayout
import com.rvcinemaview.R
import com.rvcinemaview.data.api.ApiClient
import com.rvcinemaview.data.api.MediaItem
import com.rvcinemaview.data.repository.LibraryRepository
import com.rvcinemaview.ui.common.GridConfig
import com.rvcinemaview.ui.common.GridItem
import com.rvcinemaview.ui.common.MediaGridAdapter
import com.rvcinemaview.ui.common.SkeletonAdapter
import com.rvcinemaview.ui.connect.ConnectActivity
import com.rvcinemaview.ui.player.PlayerActivity
import com.rvcinemaview.ui.settings.SettingsActivity
import com.rvcinemaview.ui.tv.TvPlayerActivity
import com.rvcinemaview.util.DeviceUtils
import kotlinx.coroutines.launch

/**
 * Unified browse activity that works on all device types.
 * Loads entire library structure once and navigates using cached data.
 */
/**
 * Breadcrumb item for navigation path display
 */
data class BreadcrumbItem(
    val id: String?,
    val name: String
)

class UnifiedBrowseActivity : FragmentActivity() {

    companion object {
        private const val KEY_CURRENT_FOLDER_ID = "current_folder_id"
        private const val KEY_BREADCRUMB_IDS = "breadcrumb_ids"
        private const val KEY_BREADCRUMB_NAMES = "breadcrumb_names"
    }

    private lateinit var recyclerView: RecyclerView
    private lateinit var backButton: ImageButton
    private lateinit var settingsButton: ImageButton
    private lateinit var progressBar: ProgressBar
    private lateinit var statusText: TextView
    private lateinit var breadcrumbsContainer: LinearLayout
    private lateinit var breadcrumbsScroll: HorizontalScrollView
    private lateinit var shimmerLayout: ShimmerFrameLayout
    private lateinit var skeletonRecyclerView: RecyclerView
    private lateinit var adapter: MediaGridAdapter

    private val repository = LibraryRepository.getInstance()
    private val breadcrumbs = mutableListOf<BreadcrumbItem>()
    private var currentFolderId: String? = null
    private var isTV: Boolean = false
    private var isInitialLoad = true

    private val backPressedCallback = object : OnBackPressedCallback(true) {
        override fun handleOnBackPressed() {
            if (breadcrumbs.size > 1) {
                // Remove current folder from breadcrumbs
                breadcrumbs.removeAt(breadcrumbs.lastIndex)
                // Navigate to previous folder
                currentFolderId = breadcrumbs.lastOrNull()?.id
                displayCurrentFolder()
            } else {
                isEnabled = false
                onBackPressedDispatcher.onBackPressed()
            }
        }
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_browse_unified)

        isTV = DeviceUtils.isTV(this)

        if (!ApiClient.isInitialized()) {
            navigateToConnect()
            return
        }

        initViews()
        setupRecyclerView()
        onBackPressedDispatcher.addCallback(this, backPressedCallback)

        // Restore state if available
        savedInstanceState?.let { state ->
            currentFolderId = state.getString(KEY_CURRENT_FOLDER_ID)
            val ids = state.getStringArrayList(KEY_BREADCRUMB_IDS)
            val names = state.getStringArrayList(KEY_BREADCRUMB_NAMES)
            if (ids != null && names != null && ids.size == names.size) {
                breadcrumbs.clear()
                for (i in ids.indices) {
                    breadcrumbs.add(BreadcrumbItem(ids[i], names[i]))
                }
            }
        }

        // Load library if not already loaded
        if (repository.isLoaded()) {
            // Initialize breadcrumbs if empty (e.g., fresh start)
            if (breadcrumbs.isEmpty()) {
                val libraryName = repository.getLibraryName().ifEmpty { getString(R.string.app_name) }
                breadcrumbs.add(BreadcrumbItem(null, libraryName))
            }
            displayCurrentFolder()
        } else {
            loadFullLibrary()
        }
    }

    override fun onSaveInstanceState(outState: Bundle) {
        super.onSaveInstanceState(outState)
        outState.putString(KEY_CURRENT_FOLDER_ID, currentFolderId)
        outState.putStringArrayList(KEY_BREADCRUMB_IDS, ArrayList(breadcrumbs.map { it.id }))
        outState.putStringArrayList(KEY_BREADCRUMB_NAMES, ArrayList(breadcrumbs.map { it.name }))
    }

    override fun onResume() {
        super.onResume()

        // Skip if initial load (onCreate handles it)
        if (isInitialLoad) {
            isInitialLoad = false
            return
        }

        // Only refresh progress, then redisplay current folder
        if (repository.isLoaded()) {
            lifecycleScope.launch {
                repository.refreshProgress()
                displayCurrentFolder()
            }
        }
    }

    override fun onConfigurationChanged(newConfig: Configuration) {
        super.onConfigurationChanged(newConfig)
        updateGridColumns()
    }

    private fun initViews() {
        recyclerView = findViewById(R.id.recyclerView)
        backButton = findViewById(R.id.backButton)
        settingsButton = findViewById(R.id.settingsButton)
        progressBar = findViewById(R.id.progressBar)
        statusText = findViewById(R.id.statusText)
        breadcrumbsContainer = findViewById(R.id.breadcrumbsContainer)
        breadcrumbsScroll = findViewById(R.id.breadcrumbsScroll)
        shimmerLayout = findViewById(R.id.shimmerLayout)
        skeletonRecyclerView = findViewById(R.id.skeletonRecyclerView)

        backButton.setOnClickListener {
            backPressedCallback.handleOnBackPressed()
        }

        settingsButton.setOnClickListener {
            startActivity(Intent(this, SettingsActivity::class.java))
        }

        // Setup skeleton RecyclerView
        setupSkeletonRecyclerView()
    }

    private fun setupSkeletonRecyclerView() {
        val columns = GridConfig.getColumnCount(this)
        skeletonRecyclerView.layoutManager = GridLayoutManager(this, columns)
        skeletonRecyclerView.adapter = SkeletonAdapter(8)

        val padding = resources.getDimensionPixelSize(R.dimen.grid_padding)
        skeletonRecyclerView.setPadding(padding, padding, padding, padding)
        skeletonRecyclerView.clipToPadding = false
    }

    private fun setupRecyclerView() {
        adapter = MediaGridAdapter(
            onFolderClick = { id, name -> navigateToFolder(id, name) },
            onMediaClick = { media -> playMedia(media) }
        )

        val columns = GridConfig.getColumnCount(this)
        recyclerView.layoutManager = GridLayoutManager(this, columns)
        recyclerView.adapter = adapter

        val padding = resources.getDimensionPixelSize(R.dimen.grid_padding)
        recyclerView.setPadding(padding, padding, padding, padding)
        recyclerView.clipToPadding = false
        recyclerView.setHasFixedSize(true)
    }

    private fun updateGridColumns() {
        val columns = GridConfig.getColumnCount(this)
        (recyclerView.layoutManager as? GridLayoutManager)?.spanCount = columns
    }

    /**
     * Load entire library structure (called once at startup)
     */
    private fun loadFullLibrary() {
        showLoading()

        lifecycleScope.launch {
            val result = repository.loadLibrary()

            if (result.isSuccess) {
                // Initialize breadcrumbs with library name from server
                val libraryName = repository.getLibraryName().ifEmpty { getString(R.string.app_name) }
                breadcrumbs.clear()
                breadcrumbs.add(BreadcrumbItem(null, libraryName))
                displayCurrentFolder()
            } else {
                showError(result.exceptionOrNull()?.localizedMessage
                    ?: getString(R.string.error_loading_library))
            }
        }
    }

    /**
     * Display current folder from cache (instant, no network)
     */
    private fun displayCurrentFolder() {
        lifecycleScope.launch {
            val content = if (currentFolderId == null) {
                repository.getRootContent()
            } else {
                repository.getFolderContent(currentFolderId!!)
            }

            if (content == null) {
                showError(getString(R.string.error_loading_folder))
                return@launch
            }

            // Build grid items
            val items = mutableListOf<GridItem>()

            content.folders.forEach { folder ->
                items.add(GridItem.FolderItem(id = folder.id, name = folder.name))
            }

            content.mediaItems.forEach { mediaWithProgress ->
                items.add(GridItem.MediaItemData(mediaWithProgress.media, mediaWithProgress.progress))
            }

            showContent(items)
        }
    }

    private fun navigateToFolder(folderId: String, folderName: String) {
        breadcrumbs.add(BreadcrumbItem(folderId, folderName))
        currentFolderId = folderId
        displayCurrentFolder()
    }

    /**
     * Navigate to a specific breadcrumb (for clicking on breadcrumb items)
     */
    private fun navigateToBreadcrumb(index: Int) {
        if (index < 0 || index >= breadcrumbs.size - 1) return

        // Remove all breadcrumbs after the clicked one
        while (breadcrumbs.size > index + 1) {
            breadcrumbs.removeAt(breadcrumbs.lastIndex)
        }

        currentFolderId = breadcrumbs[index].id
        displayCurrentFolder()
    }

    /**
     * Update breadcrumbs UI
     */
    private fun updateBreadcrumbsUI() {
        breadcrumbsContainer.removeAllViews()

        breadcrumbs.forEachIndexed { index, item ->
            // Add separator if not first item
            if (index > 0) {
                val separator = TextView(this).apply {
                    text = "›"
                    textSize = 18f
                    setTextColor(ContextCompat.getColor(context, R.color.text_hint))
                    setPadding(16, 0, 16, 0)
                }
                breadcrumbsContainer.addView(separator)
            }

            // Add breadcrumb text
            val textView = TextView(this).apply {
                text = item.name
                textSize = if (index == breadcrumbs.lastIndex) 20f else 16f
                setTextColor(
                    ContextCompat.getColor(
                        context,
                        if (index == breadcrumbs.lastIndex) R.color.text_primary else R.color.text_secondary
                    )
                )
                if (index == breadcrumbs.lastIndex) {
                    typeface = Typeface.DEFAULT_BOLD
                }
                // Make clickable if not the current (last) item
                if (index < breadcrumbs.lastIndex) {
                    isClickable = true
                    isFocusable = true
                    setBackgroundResource(R.drawable.header_button_background)
                    setPadding(8, 4, 8, 4)
                    setOnClickListener { navigateToBreadcrumb(index) }
                }
            }
            breadcrumbsContainer.addView(textView)
        }

        // Scroll to end to show current folder
        breadcrumbsScroll.post {
            breadcrumbsScroll.fullScroll(HorizontalScrollView.FOCUS_RIGHT)
        }
    }

    private fun playMedia(media: MediaItem) {
        val playerClass = if (isTV) {
            TvPlayerActivity::class.java
        } else {
            PlayerActivity::class.java
        }

        val intent = Intent(this, playerClass).apply {
            putExtra(PlayerActivity.EXTRA_MEDIA_ID, media.id)
            putExtra(PlayerActivity.EXTRA_MEDIA_TITLE, media.title)
        }
        startActivity(intent)
    }

    private fun showLoading() {
        // Hide main content
        recyclerView.visibility = View.GONE
        statusText.visibility = View.GONE
        progressBar.visibility = View.GONE
        backButton.visibility = View.GONE

        // Show shimmer skeleton loading
        shimmerLayout.visibility = View.VISIBLE
        shimmerLayout.startShimmer()

        // Show loading text in breadcrumbs area
        breadcrumbsContainer.removeAllViews()
        val loadingText = TextView(this).apply {
            text = getString(R.string.loading)
            textSize = 20f
            setTextColor(ContextCompat.getColor(context, R.color.text_secondary))
        }
        breadcrumbsContainer.addView(loadingText)
    }

    private fun showContent(items: List<GridItem>) {
        // Hide loading states
        progressBar.visibility = View.GONE
        statusText.visibility = View.GONE
        shimmerLayout.stopShimmer()
        shimmerLayout.visibility = View.GONE

        // Update breadcrumbs UI
        updateBreadcrumbsUI()

        // Update back button visibility
        val canGoBack = breadcrumbs.size > 1
        backButton.visibility = if (canGoBack) View.VISIBLE else View.GONE

        if (items.isEmpty()) {
            recyclerView.visibility = View.GONE
            statusText.text = getString(R.string.no_media)
            statusText.visibility = View.VISIBLE
        } else {
            recyclerView.visibility = View.VISIBLE
            adapter.submitList(items) {
                // Focus on first item after list is rendered
                recyclerView.post {
                    recyclerView.getChildAt(0)?.requestFocus()
                }
            }
        }
    }

    private fun showError(message: String) {
        progressBar.visibility = View.GONE
        recyclerView.visibility = View.GONE
        shimmerLayout.stopShimmer()
        shimmerLayout.visibility = View.GONE
        statusText.text = message
        statusText.visibility = View.VISIBLE
        backButton.visibility = View.GONE
    }

    private fun navigateToConnect() {
        val intent = Intent(this, ConnectActivity::class.java)
        intent.flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK
        startActivity(intent)
        finish()
    }
}
