package com.rvcinemaview.ui.common

import android.graphics.Color
import android.graphics.drawable.GradientDrawable
import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import android.widget.ImageView
import android.widget.ProgressBar
import android.widget.TextView
import androidx.core.content.ContextCompat
import androidx.recyclerview.widget.DiffUtil
import androidx.recyclerview.widget.ListAdapter
import androidx.recyclerview.widget.RecyclerView
import coil.request.ImageRequest
import com.google.android.material.card.MaterialCardView
import com.rvcinemaview.R
import com.rvcinemaview.data.api.ApiClient
import com.rvcinemaview.data.api.MediaItem

/**
 * Unified grid item types for folders and media.
 */
sealed class GridItem {
    data class FolderItem(
        val id: String,
        val name: String
    ) : GridItem()

    data class MediaItemData(
        val media: MediaItem,
        val progress: Double = 0.0 // 0.0 - 1.0
    ) : GridItem()
}

/**
 * Unified adapter for displaying folders and media items in a grid layout.
 * Works across phone, tablet, and TV (via RecyclerView on TV if needed).
 */
class MediaGridAdapter(
    private val onFolderClick: (String, String) -> Unit, // id, name
    private val onMediaClick: (MediaItem) -> Unit
) : ListAdapter<GridItem, RecyclerView.ViewHolder>(GridDiffCallback()) {

    companion object {
        private const val VIEW_TYPE_FOLDER = 0
        private const val VIEW_TYPE_MEDIA = 1
    }

    override fun getItemViewType(position: Int): Int {
        return when (getItem(position)) {
            is GridItem.FolderItem -> VIEW_TYPE_FOLDER
            is GridItem.MediaItemData -> VIEW_TYPE_MEDIA
        }
    }

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): RecyclerView.ViewHolder {
        val inflater = LayoutInflater.from(parent.context)
        return when (viewType) {
            VIEW_TYPE_FOLDER -> FolderViewHolder(
                inflater.inflate(R.layout.item_folder_card, parent, false)
            )
            else -> MediaViewHolder(
                inflater.inflate(R.layout.item_media_card, parent, false)
            )
        }
    }

    override fun onBindViewHolder(holder: RecyclerView.ViewHolder, position: Int) {
        when (val item = getItem(position)) {
            is GridItem.FolderItem -> (holder as FolderViewHolder).bind(item, onFolderClick)
            is GridItem.MediaItemData -> (holder as MediaViewHolder).bind(item.media, item.progress, onMediaClick)
        }
    }

    /**
     * ViewHolder for folder cards.
     */
    class FolderViewHolder(itemView: View) : RecyclerView.ViewHolder(itemView) {
        private val card: MaterialCardView = itemView as MaterialCardView
        private val titleText: TextView = itemView.findViewById(R.id.titleText)

        init {
            // Setup focus handling once in init
            setupFocusHandling(card)
        }

        fun bind(folder: GridItem.FolderItem, onClick: (String, String) -> Unit) {
            titleText.text = folder.name
            card.setOnClickListener { onClick(folder.id, folder.name) }
        }
    }

    /**
     * ViewHolder for media cards.
     */
    class MediaViewHolder(itemView: View) : RecyclerView.ViewHolder(itemView) {
        private val card: MaterialCardView = itemView as MaterialCardView
        private val thumbnailView: ImageView = itemView.findViewById(R.id.thumbnailView)
        private val iconOverlay: ImageView = itemView.findViewById(R.id.iconOverlay)
        private val qualityBadge: TextView = itemView.findViewById(R.id.qualityBadge)
        private val durationBadge: TextView = itemView.findViewById(R.id.durationBadge)
        private val progressBar: ProgressBar = itemView.findViewById(R.id.progressBar)
        private val titleText: TextView = itemView.findViewById(R.id.titleText)
        private val subtitleText: TextView = itemView.findViewById(R.id.subtitleText)

        init {
            // Setup focus handling once in init
            setupFocusHandling(card)
        }

        fun bind(media: MediaItem, progress: Double, onClick: (MediaItem) -> Unit) {
            val context = itemView.context
            titleText.text = media.title

            // Quality badge with color coding
            if (media.height != null) {
                val (qualityText, textColor, bgDrawable) = when {
                    media.height >= 2160 -> Triple("4K", R.color.quality_4k, R.drawable.quality_badge_4k)
                    media.height >= 1080 -> Triple("1080p", R.color.quality_1080p, R.drawable.quality_badge_1080p)
                    media.height >= 720 -> Triple("720p", R.color.quality_720p, R.drawable.quality_badge_720p)
                    else -> Triple("SD", R.color.quality_sd, R.drawable.quality_badge_sd)
                }
                qualityBadge.text = qualityText
                qualityBadge.setTextColor(ContextCompat.getColor(context, textColor))
                qualityBadge.setBackgroundResource(bgDrawable)
                qualityBadge.visibility = View.VISIBLE
            } else {
                qualityBadge.visibility = View.GONE
            }

            // Build subtitle (size only now, quality is in badge)
            subtitleText.text = formatFileSize(media.size)
            subtitleText.visibility = View.VISIBLE

            // Duration badge
            media.duration?.let { seconds ->
                durationBadge.text = formatDuration(seconds)
                durationBadge.visibility = View.VISIBLE
            } ?: run {
                durationBadge.visibility = View.GONE
            }

            // Always load thumbnail
            loadThumbnail(media)

            // Progress bar (show if progress > 2%)
            if (progress > 0.02 && progress < 0.95) {
                progressBar.progress = (progress * 100).toInt()
                progressBar.visibility = View.VISIBLE
            } else {
                progressBar.visibility = View.GONE
            }

            card.setOnClickListener { onClick(media) }
        }

        private fun loadThumbnail(media: MediaItem) {
            val context = thumbnailView.context
            val thumbnailUrl = ApiClient.getThumbnailUrl(media.id)

            // Reset state - show play icon centered, full opacity
            thumbnailView.setImageDrawable(null)
            iconOverlay.visibility = View.VISIBLE
            iconOverlay.alpha = 1.0f

            val request = ImageRequest.Builder(context)
                .data(thumbnailUrl)
                .crossfade(true)
                .target(
                    onStart = {
                        // Show play icon while loading (full opacity)
                        iconOverlay.visibility = View.VISIBLE
                        iconOverlay.alpha = 1.0f
                    },
                    onSuccess = { result ->
                        thumbnailView.setImageDrawable(result)
                        // Fade play icon when thumbnail loaded (semi-transparent)
                        iconOverlay.visibility = View.VISIBLE
                        iconOverlay.alpha = 0.7f
                    },
                    onError = {
                        // Keep play icon visible on error (full opacity)
                        thumbnailView.setImageDrawable(null)
                        iconOverlay.visibility = View.VISIBLE
                        iconOverlay.alpha = 1.0f
                    }
                )
                .build()

            ApiClient.getImageLoader(context).enqueue(request)
        }

        private fun formatFileSize(bytes: Long): String {
            return when {
                bytes >= 1_073_741_824 -> String.format("%.1f GB", bytes / 1_073_741_824.0)
                bytes >= 1_048_576 -> String.format("%.0f MB", bytes / 1_048_576.0)
                else -> String.format("%.0f KB", bytes / 1024.0)
            }
        }

        private fun formatDuration(seconds: Long): String {
            val hours = seconds / 3600
            val minutes = (seconds % 3600) / 60
            val secs = seconds % 60

            return if (hours > 0) {
                String.format("%d:%02d:%02d", hours, minutes, secs)
            } else {
                String.format("%d:%02d", minutes, secs)
            }
        }
    }

    /**
     * DiffUtil callback for efficient list updates.
     */
    class GridDiffCallback : DiffUtil.ItemCallback<GridItem>() {
        override fun areItemsTheSame(oldItem: GridItem, newItem: GridItem): Boolean {
            return when {
                oldItem is GridItem.FolderItem && newItem is GridItem.FolderItem ->
                    oldItem.id == newItem.id
                oldItem is GridItem.MediaItemData && newItem is GridItem.MediaItemData ->
                    oldItem.media.id == newItem.media.id
                else -> false
            }
        }

        override fun areContentsTheSame(oldItem: GridItem, newItem: GridItem): Boolean {
            return when {
                oldItem is GridItem.MediaItemData && newItem is GridItem.MediaItemData ->
                    oldItem.media == newItem.media && oldItem.progress == newItem.progress
                else -> oldItem == newItem
            }
        }
    }
}

/**
 * Setup focus handling for TV D-Pad navigation.
 * Adds visual feedback when card is focused.
 */
private fun setupFocusHandling(card: MaterialCardView) {
    val context = card.context
    val defaultElevation = card.cardElevation
    val focusedElevation = defaultElevation + 8f
    val defaultStrokeWidth = card.strokeWidth
    val focusedStrokeWidth = 4

    card.isFocusable = true
    // Don't use isFocusableInTouchMode - it causes double-click issue on TV

    card.onFocusChangeListener = View.OnFocusChangeListener { _, hasFocus ->
        if (hasFocus) {
            // Focused state - elevate and add border
            card.cardElevation = focusedElevation
            card.strokeWidth = focusedStrokeWidth
            card.strokeColor = ContextCompat.getColor(context, R.color.primary)
            card.animate()
                .scaleX(1.05f)
                .scaleY(1.05f)
                .setDuration(150)
                .start()
        } else {
            // Unfocused state - restore defaults
            card.cardElevation = defaultElevation
            card.strokeWidth = defaultStrokeWidth
            card.strokeColor = Color.TRANSPARENT
            card.animate()
                .scaleX(1.0f)
                .scaleY(1.0f)
                .setDuration(150)
                .start()
        }
    }
}
