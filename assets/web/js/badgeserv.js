// Attach listeners to get the dynamic badge generation we want

async function updateBadge(type) {
    const data = new FormData(event.currentTarget);
    const value = Object.fromEntries(data.entries());
    const queryString = $.param(value);

    const baseUrl = window.location.origin;
    const imgUrl = baseUrl + "/api/v1/badge/" + type + "?" + queryString

    const badge = document.createElement("img");
    badge.src = imgUrl
    document.getElementById(type + "-result").replaceChildren(badge);
}

document.getElementById("static-badges")
    .addEventListener('change', async (e) => {
        updateBadge("static")
}, false);

document.getElementById("static-badges")
    .addEventListener('submit', async (e) => {
        e.preventDefault(); updateBadge("static")
    }, false);

document.getElementById("dynamic-badges").
    addEventListener('change', async (e) => {
        updateBadge("dynamic");
}, false);

document.getElementById("dynamic-badges").
    addEventListener('submit', async (e) => {
        e.preventDefault(); updateBadge("dynamic");
}, false);