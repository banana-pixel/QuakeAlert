package id.web.quakealert

/**
 * The project's public URLs, defined exactly once (D-032).
 *
 * The repository URL used to live as separate private constants in the About
 * overlay and the onboarding flow, which is how the two pages silently
 * disagreed after the repository moved. Feature code references these; nothing
 * else defines a project URL.
 */
object ProjectLinks {
    /** Source repository — onboarding bug-report link and About "GitHub" action. */
    const val REPO_URL = "https://github.com/banana-pixel/QuakeAlert"

    /** Author profile — onboarding credit link. */
    const val PROFILE_URL = "https://github.com/banana-pixel"
}
