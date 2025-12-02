package com.rvcinemaview.ui.tv

import android.content.Context
import android.os.Bundle
import android.view.KeyEvent
import android.view.View
import android.view.WindowManager
import androidx.core.content.ContextCompat
import androidx.fragment.app.FragmentActivity
import androidx.leanback.app.VideoSupportFragment
import androidx.leanback.app.VideoSupportFragmentGlueHost
import androidx.leanback.media.PlaybackTransportControlGlue
import androidx.leanback.widget.Action
import androidx.leanback.widget.ArrayObjectAdapter
import androidx.leanback.widget.PlaybackControlsRow
import androidx.lifecycle.lifecycleScope
import androidx.media3.common.MediaItem
import androidx.media3.exoplayer.ExoPlayer
import androidx.media3.ui.leanback.LeanbackPlayerAdapter
import com.rvcinemaview.R
import com.rvcinemaview.data.api.ApiClient
import com.rvcinemaview.ui.player.PlaybackHelper

class TvPlayerActivity : FragmentActivity() {

    companion object {
        const val EXTRA_MEDIA_ID = "media_id"
        const val EXTRA_MEDIA_TITLE = "media_title"
        const val EXTRA_START_POSITION = "start_position"
    }

    private var playerFragment: TvPlayerFragment? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)

        // Keep screen on during playback
        window.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON)

        val mediaId = intent.getStringExtra(EXTRA_MEDIA_ID) ?: run {
            finish()
            return
        }
        val mediaTitle = intent.getStringExtra(EXTRA_MEDIA_TITLE) ?: getString(R.string.video)
        val startPosition = intent.getLongExtra(EXTRA_START_POSITION, 0L)

        if (savedInstanceState == null) {
            playerFragment = TvPlayerFragment().apply {
                arguments = Bundle().apply {
                    putString(EXTRA_MEDIA_ID, mediaId)
                    putString(EXTRA_MEDIA_TITLE, mediaTitle)
                    putLong(EXTRA_START_POSITION, startPosition)
                }
            }
            supportFragmentManager.beginTransaction()
                .replace(android.R.id.content, playerFragment!!)
                .commit()
        } else {
            playerFragment = supportFragmentManager.findFragmentById(android.R.id.content) as? TvPlayerFragment
        }
    }

    override fun onKeyDown(keyCode: Int, event: KeyEvent?): Boolean {
        return playerFragment?.handleKeyEvent(keyCode, event) ?: super.onKeyDown(keyCode, event)
    }

    @Deprecated("Deprecated in Java")
    override fun onBackPressed() {
        playerFragment?.onBackPressed()
        super.onBackPressed()
    }
}

class TvPlayerFragment : VideoSupportFragment() {

    private var player: ExoPlayer? = null
    private var playerGlue: VideoPlayerGlue? = null
    private var mediaId: String? = null
    private var lastSavedPosition: Long = 0L

    companion object {
        private const val SEEK_STEP_MS = 10_000L      // 10 seconds
        private const val SEEK_FAST_STEP_MS = 60_000L // 1 minute
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        initializePlayer()
    }

    override fun onViewCreated(view: View, savedInstanceState: Bundle?) {
        super.onViewCreated(view, savedInstanceState)
        view.setBackgroundColor(ContextCompat.getColor(requireContext(), R.color.black))
    }

    private fun initializePlayer() {
        mediaId = arguments?.getString(TvPlayerActivity.EXTRA_MEDIA_ID) ?: return
        val mediaTitle = arguments?.getString(TvPlayerActivity.EXTRA_MEDIA_TITLE)
            ?: getString(R.string.video)
        val startPosition = arguments?.getLong(TvPlayerActivity.EXTRA_START_POSITION, 0L) ?: 0L

        player = ExoPlayer.Builder(requireContext())
            .setSeekBackIncrementMs(SEEK_STEP_MS)
            .setSeekForwardIncrementMs(SEEK_STEP_MS)
            .build()

        val playerAdapter = LeanbackPlayerAdapter(requireContext(), player!!, 16)
        playerGlue = VideoPlayerGlue(requireContext(), playerAdapter).apply {
            host = VideoSupportFragmentGlueHost(this@TvPlayerFragment)
            title = mediaTitle
            isSeekEnabled = true

            // Set up action callbacks
            setOnActionCallback { action ->
                when (action.id) {
                    VideoPlayerGlue.ACTION_REWIND -> seekBy(-SEEK_FAST_STEP_MS)
                    VideoPlayerGlue.ACTION_FAST_FORWARD -> seekBy(SEEK_FAST_STEP_MS)
                }
            }
        }

        val streamUrl = ApiClient.getStreamUrl(mediaId!!)
        val mediaItem = MediaItem.fromUri(streamUrl)

        player?.apply {
            setMediaItem(mediaItem)
            prepare()

            // Seek to start position if resuming (position is in seconds, convert to ms)
            if (startPosition > 0) {
                seekTo(startPosition * 1000)
                playWhenReady = true
            } else {
                // Load saved position from server, then start playback
                loadSavedPosition(this)
            }
        }
    }

    private fun loadSavedPosition(exoPlayer: ExoPlayer) {
        val id = mediaId ?: return
        PlaybackHelper.loadSavedPosition(lifecycleScope, id, exoPlayer)
        exoPlayer.playWhenReady = true
    }

    fun handleKeyEvent(keyCode: Int, event: KeyEvent?): Boolean {
        if (event?.action != KeyEvent.ACTION_DOWN) return false

        return when (keyCode) {
            // Back button - stop and exit
            KeyEvent.KEYCODE_BACK -> {
                onBackPressed()
                activity?.finish()
                true
            }
            // Play/Pause toggle
            KeyEvent.KEYCODE_MEDIA_PLAY_PAUSE,
            KeyEvent.KEYCODE_DPAD_CENTER -> {
                togglePlayPause()
                true
            }
            // Seek forward
            KeyEvent.KEYCODE_DPAD_RIGHT,
            KeyEvent.KEYCODE_MEDIA_FAST_FORWARD -> {
                val step = if (keyCode == KeyEvent.KEYCODE_MEDIA_FAST_FORWARD)
                    SEEK_FAST_STEP_MS else SEEK_STEP_MS
                seekBy(step)
                true
            }
            // Seek backward
            KeyEvent.KEYCODE_DPAD_LEFT,
            KeyEvent.KEYCODE_MEDIA_REWIND -> {
                val step = if (keyCode == KeyEvent.KEYCODE_MEDIA_REWIND)
                    SEEK_FAST_STEP_MS else SEEK_STEP_MS
                seekBy(-step)
                true
            }
            // Stop
            KeyEvent.KEYCODE_MEDIA_STOP -> {
                onBackPressed()
                activity?.finish()
                true
            }
            else -> false
        }
    }

    fun onBackPressed() {
        savePlaybackPosition()
        player?.stop()
        player?.release()
        player = null
    }

    private fun savePlaybackPosition() {
        val id = mediaId ?: return
        PlaybackHelper.savePlaybackPosition(lifecycleScope, id, player)
    }

    private fun togglePlayPause() {
        player?.let {
            if (it.isPlaying) {
                it.pause()
            } else {
                it.play()
            }
        }
    }

    private fun seekBy(deltaMs: Long) {
        player?.let {
            val newPosition = (it.currentPosition + deltaMs).coerceIn(0, it.duration)
            it.seekTo(newPosition)
        }
    }

    override fun onPause() {
        super.onPause()
        savePlaybackPosition()
        player?.pause()
    }

    override fun onDestroy() {
        super.onDestroy()
        player?.release()
        player = null
    }
}

/**
 * Custom PlaybackTransportControlGlue with rewind/fast-forward actions.
 */
class VideoPlayerGlue(
    context: Context,
    playerAdapter: LeanbackPlayerAdapter
) : PlaybackTransportControlGlue<LeanbackPlayerAdapter>(context, playerAdapter) {

    companion object {
        const val ACTION_REWIND = 1L
        const val ACTION_FAST_FORWARD = 2L
    }

    private val rewindAction = PlaybackControlsRow.RewindAction(context)
    private val fastForwardAction = PlaybackControlsRow.FastForwardAction(context)

    private var actionCallback: ((Action) -> Unit)? = null

    fun setOnActionCallback(callback: (Action) -> Unit) {
        actionCallback = callback
    }

    override fun onCreatePrimaryActions(primaryActionsAdapter: ArrayObjectAdapter) {
        super.onCreatePrimaryActions(primaryActionsAdapter)

        // Add rewind before play/pause
        primaryActionsAdapter.add(0, rewindAction)
        // Add fast forward after play/pause (which is at index 1 after adding rewind)
        primaryActionsAdapter.add(fastForwardAction)
    }

    override fun onActionClicked(action: Action) {
        when (action) {
            rewindAction -> actionCallback?.invoke(action.apply { id = ACTION_REWIND })
            fastForwardAction -> actionCallback?.invoke(action.apply { id = ACTION_FAST_FORWARD })
            else -> super.onActionClicked(action)
        }
    }
}
