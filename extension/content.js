// Helper to get video ID
function getVideoId() {
    try {
        const u = new URL(location.href);
        const v = u.searchParams.get("v");
        if (v) return v;
        // handle youtu.be/ID and path-based URLs (/shorts/ID, /embed/ID)
        const host = u.hostname.toLowerCase();
        const path = u.pathname || "";
        if (host.includes("youtu.be")) {
            const id = path.replace(/^\/+|\/+$/g, "");
            return id || null;
        }
        const segs = path.split("/").filter(Boolean);
        if (segs.length > 0) {
            const last = segs[segs.length - 1];
            // if last segment looks like a video id (11 chars) return it
            if (/^[a-zA-Z0-9_-]{11}$/.test(last)) return last;
        }
        return null;
    } catch (e) {
        return null;
    }
}

// Track pending injectUI retries so we can cancel stale ones
let _injectRetryTimer = null;

// Initial Setup
function init() {
    console.log("[R2C] Init called");
    const videoId = getVideoId();
    console.log("[R2C] Video ID:", videoId);
    if (!videoId) return;

    // Cancel any pending injectUI retry from a previous init() call
    if (_injectRetryTimer) {
        clearTimeout(_injectRetryTimer);
        _injectRetryTimer = null;
    }

    injectUI(videoId);
    fetchComments(videoId);
}

// Inject UI logic
function injectUI(videoId) {
    // Remove existing if any
    const existing = document.getElementById("r2c-container");
    if (existing) existing.remove();

    // Find insertion point
    // We try to find the native comments section
    const target = document.querySelector("#comments") || document.querySelector("ytd-comments");

    if (!target) {
        // Retry if DOM not ready (use tracked timer so init() can cancel it)
        _injectRetryTimer = setTimeout(() => injectUI(videoId), 1000);
        return;
    }

    // Create Container
    const container = document.createElement("div");
    container.id = "r2c-container";
    container.style.border = "2px solid red"; // DEBUG: Visual indicator

    // Header
    const header = document.createElement("div");
    header.id = "r2c-header";
    const headerTitle = document.createTextNode("Right2Comment ");
    header.appendChild(headerTitle);
    container.appendChild(header);

    // Comment List
    const commentList = document.createElement("div");
    commentList.id = "r2c-comment-list";
    const loadingMeta = document.createElement("div");
    loadingMeta.className = "r2c-comment-meta";
    loadingMeta.textContent = "Loading...";
    commentList.appendChild(loadingMeta);
    container.appendChild(commentList);

    // Form
    const form = document.createElement("form");
    form.id = "r2c-form";

    const input = document.createElement("input");
    input.type = "text";
    input.id = "r2c-input";
    input.placeholder = "Add a public comment...";
    input.autocomplete = "off";

    const btn = document.createElement("button");
    btn.type = "submit";
    btn.id = "r2c-submit";
    btn.disabled = true;
    btn.textContent = "Comment";

    form.appendChild(input);
    form.appendChild(btn);
    container.appendChild(form);

    // Insert above native comments — guard if parent is missing
    if (target.parentNode) {
        target.parentNode.insertBefore(container, target);
    } else {
        // Fallback: append inside target or to body
        try { target.appendChild(container); } catch (e) { document.body.appendChild(container); }
    }

    // Event Listeners
    input.addEventListener("input", () => {
        btn.disabled = input.value.trim() === "";
    });

    form.addEventListener("submit", async (e) => {
        e.preventDefault();
        const text = input.value.trim();
        if (!text) return;

        btn.disabled = true;
        btn.textContent = "Posting...";

        try {
            await postComment(videoId, text);
            input.value = "";
            // Refresh comments
            await fetchComments(videoId);
        } catch (err) {
            alert("Failed to post comment: " + err.message);
        } finally {
            btn.disabled = false;
            btn.textContent = "Comment";
        }
    });
}

// Fetch Comments
async function fetchComments(videoId) {
    const list = document.getElementById("r2c-comment-list");
    if (!list) return;

    try {
        console.log("[R2C] Requesting comments for", videoId);
        const response = await sendMessageAsync({
            action: "FETCH_COMMENTS",
            videoId: videoId
        });
        console.log("[R2C] Response:", response);

        if (!response.success) {
            throw new Error(response.error);
        }

        renderComments(response.data);
    } catch (err) {
        list.innerHTML = ""; // Clear list
        const errorDiv = document.createElement("div");
        errorDiv.className = "r2c-comment-meta";
        errorDiv.textContent = "Error loading comments.";
        list.appendChild(errorDiv);
        console.error(err);
    }
}

// Post Comment
async function postComment(videoId, text) {
    const response = await sendMessageAsync({
        action: "POST_COMMENT",
        videoId: videoId,
        text: text
    });
    if (!response || !response.success) {
        throw new Error((response && response.error) || "Unknown error");
    }
}

// Render Comments
function renderComments(comments) {
    const list = document.getElementById("r2c-comment-list");
    if (!list) return;

    list.innerHTML = ""; // Safe to clear simple text/elements

    if (comments.length === 0) {
        const noComments = document.createElement("div");
        noComments.className = "r2c-comment-meta";
        noComments.textContent = "No comments yet. Be the first!";
        list.appendChild(noComments);
        return;
    }

    comments.forEach(c => {
        const el = document.createElement("div");
        el.className = "r2c-comment";

        const textDiv = document.createElement("div");
        textDiv.className = "r2c-comment-text";
        textDiv.textContent = c.text; // Safe from XSS

        const metaDiv = document.createElement("div");
        metaDiv.className = "r2c-comment-meta";
        metaDiv.textContent = c.createdAt;

        el.appendChild(textDiv);
        el.appendChild(metaDiv);
        list.appendChild(el);
    });
}

// Utils
function escapeHtml(text) {
    if (!text) return "";
    return text
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#039;");
}

// Navigation Handling for SPA
// Method 1: Listen for YouTube's custom event
document.addEventListener("yt-navigate-finish", init);

// Method 2: Listen for URL changes via MutationObserver (fallback)
// YouTube navigation sometimes doesn't fire events reliably if loaded from specialized views
let lastUrl = location.href;
new MutationObserver(() => {
    const url = location.href;
    if (url !== lastUrl) {
        lastUrl = url;
        if (getVideoId()) {
            init();
        }
    }
}).observe(document, { subtree: true, childList: true });

// Initial run
if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
} else {
    init();
}

// Add a helper to send messages that works with both Chrome (callback) and browsers that return a Promise.
function sendMessageAsync(message) {
    return new Promise((resolve, reject) => {
        try {
            // Try Chrome-style with callback
            const maybePromise = chrome.runtime.sendMessage(message, (response) => {
                if (chrome.runtime.lastError) {
                    reject(chrome.runtime.lastError);
                } else {
                    resolve(response);
                }
            });
            // Some implementations return a Promise; handle that too
            if (maybePromise && typeof maybePromise.then === "function") {
                maybePromise.then(resolve).catch(reject);
            }
        } catch (err) {
            // Fallback to browser.* if available
            if (typeof browser !== "undefined" && browser.runtime && browser.runtime.sendMessage) {
                browser.runtime.sendMessage(message).then(resolve).catch(reject);
            } else {
                reject(err);
            }
        }
    });
}
