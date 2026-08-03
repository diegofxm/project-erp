// Propuesta de diseño — comportamiento compartido entre los mockups.
(function () {
  document.addEventListener("click", (e) => {
    const toggle = e.target.closest(".topbar-toggle");
    if (toggle) {
      document.querySelector(".sidebar")?.classList.toggle("collapsed");
      return;
    }

    const groupRow = e.target.closest("tr.group-row");
    if (groupRow) {
      groupRow.classList.toggle("collapsed");
      const chevron = groupRow.querySelector(".group-chevron");
      let sib = groupRow.nextElementSibling;
      while (sib && !sib.classList.contains("group-row")) {
        sib.style.display = groupRow.classList.contains("collapsed") ? "none" : "";
        sib = sib.nextElementSibling;
      }
      if (chevron) chevron.style.transform = groupRow.classList.contains("collapsed") ? "rotate(-90deg)" : "rotate(0deg)";
      return;
    }

    const tab = e.target.closest(".notebook-tab");
    if (tab) {
      const group = tab.closest(".notebook");
      group.querySelectorAll(".notebook-tab").forEach((t) => t.classList.remove("active"));
      group.querySelectorAll(".notebook-panel").forEach((p) => p.classList.remove("active"));
      tab.classList.add("active");
      group.querySelector(`.notebook-panel[data-panel="${tab.dataset.tab}"]`)?.classList.add("active");
    }
  });
})();
