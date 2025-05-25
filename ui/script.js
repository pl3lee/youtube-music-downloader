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

	resultsDiv.innerHTML = "<p>Processing...</p>";

	try {
		const headers = {
			"Content-Type": "application/json",
		};
        if (password) {
            headers['Authorization'] = password;
        }
		const response = await fetch("/api/download", {
			method: "POST",
			headers: headers,
			body: JSON.stringify({ links: links }),
		});

		if (!response.ok) {
			let errorMsg = `Error: ${response.status} ${response.statusText}`;
			try {
				const errorData = await response.json();
				if (errorData && errorData.error) {
					errorMsg += ` - ${errorData.error}`;
				}
			} catch (e) {
				// Ignore if response is not JSON or error in parsing
			}
			resultsDiv.innerHTML = `<p style="color: red;">${errorMsg}</p>`;
			return;
		}

		const data = await response.json();

		if (data.results && data.results.length > 0) {
			let html = "<h3>Download Results:</h3><ul>";
			data.results.forEach((result) => {
				html += `<li>${result.link}: <span style="color: ${result.status === "success" ? "green" : "red"};">${result.status}</span>`;
				if (result.error) {
					html += ` - Error: ${result.error}`;
				}
				html += "</li>";
			});
			html += "</ul>";
			resultsDiv.innerHTML = html;
		} else {
			resultsDiv.innerHTML = "<p>No results returned from server.</p>";
		}
	} catch (error) {
		resultsDiv.innerHTML = `<p style="color: red;">An unexpected error occurred: ${error.message}</p>`;
		console.error("Fetch error:", error);
	}
});
