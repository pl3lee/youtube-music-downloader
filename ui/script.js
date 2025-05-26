const passwordInput = document.getElementById("pwd");
const linksTextarea = document.getElementById("links");
const submitButton = document.getElementById("submitBtn");
const resultsDiv = document.getElementById("results");

submitButton.addEventListener("click", async () => {
	const password = passwordInput.value;
	const links = linksTextarea.value
		.split("\n")
		.map((link) => link.trim())
		.filter((link) => link.length > 0);

	if (links.length === 0) {
		resultsDiv.innerHTML =
			'<p style="color: red;">Please enter at least one YouTube Music link.</p>';
		return;
	}

	resultsDiv.innerHTML = "<p>Submitting download request...</p>";
	submitButton.disabled = true;

	try {
		// Step 1: POST to /api/download to create the task
		const postHeaders = {
			"Content-Type": "application/json",
		};
		if (password) {
			postHeaders["Authorization"] = password;
		}

		const response = await fetch("/api/download", {
			method: "POST",
			headers: postHeaders,
			body: JSON.stringify({ links: links }),
		});

		if (!response.ok) {
			let errorMsg = `Error submitting task: ${response.status} ${response.statusText}`;
			try {
				const errorData = await response.json();
				if (errorData && errorData.error) {
					errorMsg += ` - ${errorData.error}`;
				}
			} catch (e) { /* Ignore if response is not JSON */ }
			resultsDiv.innerHTML = `<p style="color: red;">${errorMsg}</p>`;
			submitButton.disabled = false;
			return;
		}

		const taskData = await response.json();
		if (!taskData.task_id) {
			resultsDiv.innerHTML = `<p style="color: red;">Error: Did not receive task ID from server.</p>`;
			submitButton.disabled = false;
			return;
		}

		// Step 2: Open EventSource connection for status updates
		resultsDiv.innerHTML = `<p>Task submitted (ID: ${taskData.task_id}). Waiting for updates...</p><h3>Download Results:</h3><ul></ul>`;
		const resultsList = resultsDiv.querySelector("ul");

		// Note: Standard EventSource does not support custom headers directly.
		// Authentication for the SSE endpoint is handled by the server based on the task's creation context.
		const eventSource = new EventSource(`/api/download/status/${taskData.task_id}`);

		eventSource.onmessage = (event) => {
			// Generic message handler for 'data' events
			try {
				const result = JSON.parse(event.data);
				const listItem = document.createElement("li");
				listItem.innerHTML = `${result.link}: <span style="color: ${result.status === "success" ? "green" : "red"};">${result.status}</span>`;
				if (result.error) {
					listItem.innerHTML += ` - Error: ${result.error}`;
				}
				resultsList.appendChild(listItem);
			} catch (e) {
				console.error("Error parsing SSE data:", e, "Raw data:", event.data);
				const listItem = document.createElement("li");
				listItem.innerHTML = `<span style="color: orange;">Received non-JSON message or parse error: ${event.data}</span>`;
				resultsList.appendChild(listItem);
			}
		};

		eventSource.addEventListener('complete', (event) => {
			console.log("Received 'complete' event from server:", event.data);
			resultsDiv.innerHTML += "<p>Processing complete (server signaled completion).</p>";
			eventSource.close();
			submitButton.disabled = false;
		});

		eventSource.addEventListener('error', (event) => {
			// This 'error' event from EventSource can also be triggered if the server sends an event with 'event: error'
			if (event.data) {
				try {
					const errorDetails = JSON.parse(event.data);
					resultsDiv.innerHTML += `<p style="color: red;">Server error event: ${errorDetails.error || event.data}</p>`;
				} catch (e) {
					resultsDiv.innerHTML += `<p style="color: red;">Server error event (unparsable): ${event.data}</p>`;
				}
			}
		});


		eventSource.onerror = (error) => {
			// This handles network errors or when the connection is closed by the server without a 'complete' event.
			// If eventSource.readyState is EventSource.CLOSED, it means the connection was closed.
			// This might happen after 'complete' or due to an unhandled server-side issue.
			if (eventSource.readyState === EventSource.CLOSED) {
				console.log("EventSource connection closed by server or network error.");
				// Avoid double messaging if 'complete' was already handled.
				if (!resultsDiv.innerText.includes("Processing complete")) {
					resultsDiv.innerHTML += "<p>Connection closed. Downloads may be complete or an error occurred.</p>";
				}
			} else {
				console.error("EventSource failed:", error);
				resultsDiv.innerHTML += `<p style="color: red;">Connection error with the server. Downloads may be incomplete.</p>`;
			}
			eventSource.close(); // Ensure it's closed
			submitButton.disabled = false; // Re-enable button
		};

	} catch (error) {
		resultsDiv.innerHTML = `<p style="color: red;">An unexpected error occurred during submission: ${error.message}</p>`;
		console.error("Submission or EventSource setup error:", error);
		submitButton.disabled = false;
	}
});
