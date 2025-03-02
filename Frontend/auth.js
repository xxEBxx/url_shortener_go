document.getElementById("signup-form").addEventListener("submit", async function (event) {
    event.preventDefault();

    const username = document.getElementById("signup-username").value;
    const password = document.getElementById("signup-password").value;

    const response = await fetch("http://localhost:8080/signup", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
    });

    const message = response.ok ? "✅ Sign Up Successful!" : "❌ Sign Up Failed!";
    document.getElementById("message").innerText = message;
});

document.getElementById("login-form").addEventListener("submit", async function (event) {
    event.preventDefault();

    const username = document.getElementById("login-username").value;
    const password = document.getElementById("login-password").value;

    const response = await fetch("http://localhost:8080/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
    });
    //console.log(response);
    const data = await response.json();
    if (response.ok) {
        localStorage.setItem("jwt", data.token);
        document.getElementById("message").innerText = "✅ Login Successful!";
    } else {
        document.getElementById("message").innerText = "❌ Login Failed!";
    }
});
