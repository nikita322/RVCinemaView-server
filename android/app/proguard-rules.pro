# Add project specific ProGuard rules here.
# You can control the set of applied configuration files using the
# proguardFiles setting in build.gradle.

# Keep Retrofit
-keepattributes Signature
-keepattributes Exceptions

# Keep Gson
-keepattributes *Annotation*
-keep class com.rvcinemaview.data.api.** { *; }
