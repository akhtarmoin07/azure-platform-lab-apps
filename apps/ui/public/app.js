const products = document.querySelector("#products");
const statusBadge = document.querySelector("#status");
const template = document.querySelector("#product-template");

async function loadMedicines() {
  try {
    const response = await fetch("/api/medicines", {
      headers: { Accept: "application/json" },
    });
    if (!response.ok) {
      throw new Error(`API returned ${response.status}`);
    }

    const medicines = await response.json();
    for (const medicine of medicines) {
      const card = template.content.cloneNode(true);
      card.querySelector("h3").textContent = medicine.name;
      card.querySelector("p").textContent = medicine.description;
      card.querySelector("strong").textContent = new Intl.NumberFormat("en-DE", {
        style: "currency",
        currency: "EUR",
      }).format(medicine.price);
      products.appendChild(card);
    }
    statusBadge.textContent = "Platform healthy";
    statusBadge.classList.add("healthy");
  } catch (error) {
    statusBadge.textContent = "Backend unavailable";
    statusBadge.classList.add("unhealthy");
    products.textContent = error.message;
  }
}

loadMedicines();
