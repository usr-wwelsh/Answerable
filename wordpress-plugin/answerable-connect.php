<?php
/**
 * Plugin Name: Answerable Connect
 * Description: Makes this WordPress site agent-discoverable by connecting it to an Answerable instance. No server access, no DNS changes.
 * Version: 0.1.0
 * License: MIT
 * Author: usr-wwelsh
 */

if (!defined('ABSPATH')) {
    exit;
}

define('ANSWERABLE_CONNECT_OPTION', 'answerable_connect_instance_url');

function answerable_connect_instance_url() {
    return trim(get_option(ANSWERABLE_CONNECT_OPTION, ''));
}

const ANSWERABLE_CONNECT_PROXY_PATHS = ['llms.txt', '.well-known/agent.json', '.well-known/mcp.json'];

// Matches on the raw request URI in `init` — before WordPress's own
// rewrite/query/canonical-redirect machinery touches the request — so
// pretty-permalink trailing-slash redirects and query-var parsing can't
// interfere. Pretty permalinks (mod_rewrite routing unmatched paths to
// index.php) still have to be on for the request to reach PHP at all.
add_action('init', function () {
    $uri = parse_url($_SERVER['REQUEST_URI'] ?? '', PHP_URL_PATH);
    $path = ltrim((string) $uri, '/');
    if (in_array($path, ANSWERABLE_CONNECT_PROXY_PATHS, true)) {
        answerable_connect_proxy($path);
    }
}, 0);

// Proxies the three well-known discovery paths to the real Answerable
// instance so agents that resolve them same-origin (per convention) get a
// live answer, while Answerable itself still does all the actual work —
// this is a pass-through, not a reimplementation.
function answerable_connect_proxy($path) {
    $base = answerable_connect_instance_url();
    if (!$base) {
        status_header(404);
        header('Content-Type: text/plain; charset=utf-8');
        echo "Answerable Connect is installed but no instance URL is configured.\n";
        exit;
    }

    $upstream = rtrim($base, '/') . '/' . ltrim($path, '/');
    $response = wp_remote_get($upstream, [
        'timeout' => 5,
        'headers' => ['Accept' => '*/*'],
    ]);

    if (is_wp_error($response)) {
        status_header(502);
        header('Content-Type: text/plain; charset=utf-8');
        echo "Answerable instance unreachable.\n";
        exit;
    }

    status_header(wp_remote_retrieve_response_code($response));
    $content_type = wp_remote_retrieve_header($response, 'content-type');
    if ($content_type) {
        header('Content-Type: ' . $content_type);
    }
    echo wp_remote_retrieve_body($response);
    exit;
}

// Head/footer links mirror what Answerable's own reverse-proxy mode
// injects — pointing at this site's own (now proxied) discovery paths,
// except the MCP server endpoint, which agents can call directly on the
// Answerable instance without needing to be same-origin.
add_action('wp_head', function () {
    $base = rtrim(answerable_connect_instance_url(), '/');
    if (!$base) {
        return;
    }
    $links = [
        ['llms-txt', home_url('/llms.txt')],
        ['agent-card', home_url('/.well-known/agent.json')],
        ['mcp-manifest', home_url('/.well-known/mcp.json')],
        ['mcp-server', $base . '/mcp'],
    ];
    foreach ($links as [$rel, $href]) {
        printf('<link rel="%s" href="%s">' . "\n", esc_attr($rel), esc_url($href));
    }
});

add_action('wp_footer', function () {
    if (!answerable_connect_instance_url()) {
        return;
    }
    printf(
        '<p style="position:absolute;width:1px;height:1px;margin:-1px;padding:0;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0"><a href="%s">Agent/API data</a></p>' . "\n",
        esc_url(home_url('/llms.txt'))
    );
});

// --- Settings ---

add_action('admin_menu', function () {
    add_options_page(
        'Answerable Connect',
        'Answerable Connect',
        'manage_options',
        'answerable-connect',
        'answerable_connect_render_settings_page'
    );
});

add_action('admin_init', function () {
    register_setting('answerable_connect', ANSWERABLE_CONNECT_OPTION, [
        'type' => 'string',
        'sanitize_callback' => 'esc_url_raw',
        'default' => '',
    ]);
});

add_action('admin_notices', function () {
    if (!current_user_can('manage_options') || !answerable_connect_instance_url()) {
        return;
    }
    if (get_option('permalink_structure')) {
        return;
    }
    printf(
        '<div class="notice notice-warning"><p><strong>Answerable Connect:</strong> pretty permalinks are required for agents to reach /llms.txt. Go to <a href="%s">Settings &rarr; Permalinks</a> and choose any structure other than "Plain".</p></div>',
        esc_url(admin_url('options-permalink.php'))
    );
});

function answerable_connect_render_settings_page() {
    if (!current_user_can('manage_options')) {
        return;
    }
    ?>
    <div class="wrap">
        <h1>Answerable Connect</h1>
        <p>Point this site at your running Answerable instance. Answerable does all the work — this plugin just wires your site to it, the same way you'd install a Google Analytics or chat-widget snippet.</p>
        <form method="post" action="options.php">
            <?php settings_fields('answerable_connect'); ?>
            <table class="form-table">
                <tr>
                    <th scope="row"><label for="<?php echo esc_attr(ANSWERABLE_CONNECT_OPTION); ?>">Answerable instance URL</label></th>
                    <td>
                        <input type="url" id="<?php echo esc_attr(ANSWERABLE_CONNECT_OPTION); ?>" name="<?php echo esc_attr(ANSWERABLE_CONNECT_OPTION); ?>" value="<?php echo esc_attr(answerable_connect_instance_url()); ?>" class="regular-text" placeholder="https://myorg-answerable.up.railway.app" />
                        <p class="description">The base URL of your running Answerable instance (no trailing slash).</p>
                    </td>
                </tr>
            </table>
            <?php submit_button(); ?>
        </form>
        <?php if (answerable_connect_instance_url()): ?>
            <p>Check <a href="<?php echo esc_url(home_url('/llms.txt')); ?>" target="_blank" rel="noopener">this site's /llms.txt</a> to confirm it's live.</p>
        <?php endif; ?>
    </div>
    <?php
}
