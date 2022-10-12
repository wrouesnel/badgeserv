// Attach listeners to get the dynamic badge generation we want

async function updateStaticBadge() {
    const data = new FormData(event.currentTarget);
    const value = Object.fromEntries(data.entries());
    const queryString = $.param(value);

    const baseUrl = window.location.origin;
    const imgUrl = baseUrl + "/api/v1/badge/static?" + queryString

    const badge = document.createElement("img");
    badge.src = imgUrl
    document.getElementById("static-result").replaceChildren(badge);
}

document.getElementById("static-badges")
    .addEventListener('change', updateStaticBadge, false);

document.getElementById("static-badges")
    .addEventListener('submit', async (e) => {e.preventDefault(); updateStaticBadge()}, false);

document.getElementById("dynamic-badges").
    addEventListener('change', function() {
    var result = document.getElementById("dynamic-result");
    var currentForm = document.getElementById("dynamic-badges")
});