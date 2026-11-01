const button = document.getElementById("if-oauth-connect");
if (button) {
  button.addEventListener("click", async () => {
    button.disabled = true;
    try {
      const response = await fetch("/api/v1/oauth/infinite-flight?scope_set=admin_scopes", { credentials: "same-origin" });
      if (!response.ok) {
        throw new Error(`auth url request failed: ${response.status}`);
      }
      const payload = await response.json();
      const url = payload?.data?.url;
      if (!url) {
        throw new Error("missing authorize url");
      }
      window.location.assign(url);
    } catch (error) {
      console.error("infinite flight oauth start failed", error);
      button.disabled = false;
    }
  });
}
