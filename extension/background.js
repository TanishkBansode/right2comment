const API_URL = "https://right2comment.vercel.app";

// Listener for messages from content script
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
    if (request.action === "FETCH_COMMENTS") {
        fetchComments(request.videoId)
            .then(data => sendResponse({ success: true, data }))
            .catch(error => sendResponse({ success: false, error: error.message }));
        return true; // Keep channel open for async response
    }

    if (request.action === "POST_COMMENT") {
        postComment(request.videoId, request.text)
            .then(() => sendResponse({ success: true }))
            .catch(error => sendResponse({ success: false, error: error.message }));
        return true; // Keep channel open for async response
    }
});

async function fetchComments(videoId) {
    const res = await fetch(`${API_URL}/comments/${videoId}`);
    if (!res.ok) throw new Error("API Error: " + res.statusText);
    return await res.json();
}

async function postComment(videoId, text) {
    const formData = new FormData();
    formData.append("comment", text);

    const res = await fetch(`${API_URL}/comments/${videoId}`, {
        method: "POST",
        body: formData
    });

    if (!res.ok) {
        let errorMsg = "Unknown error";
        try {
            const err = await res.json();
            errorMsg = err.error || errorMsg;
        } catch (e) {
            // ignore JSON parse error
        }
        throw new Error(errorMsg);
    }
}
