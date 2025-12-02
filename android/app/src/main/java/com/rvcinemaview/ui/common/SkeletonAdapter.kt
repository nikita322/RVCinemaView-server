package com.rvcinemaview.ui.common

import android.view.LayoutInflater
import android.view.View
import android.view.ViewGroup
import androidx.recyclerview.widget.RecyclerView
import com.rvcinemaview.R

/**
 * Simple adapter that shows skeleton placeholder cards during loading.
 */
class SkeletonAdapter(private val itemCount: Int = 8) : RecyclerView.Adapter<SkeletonAdapter.SkeletonViewHolder>() {

    class SkeletonViewHolder(itemView: View) : RecyclerView.ViewHolder(itemView)

    override fun onCreateViewHolder(parent: ViewGroup, viewType: Int): SkeletonViewHolder {
        val view = LayoutInflater.from(parent.context)
            .inflate(R.layout.item_skeleton_card, parent, false)
        return SkeletonViewHolder(view)
    }

    override fun onBindViewHolder(holder: SkeletonViewHolder, position: Int) {
        // Nothing to bind - skeleton is static
    }

    override fun getItemCount(): Int = itemCount
}
