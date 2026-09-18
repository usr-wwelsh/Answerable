document.querySelectorAll("time[data-ts]").forEach(function (el) {
  var d = new Date(el.getAttribute("data-ts"));
  if (!isNaN(d.getTime())) {
    el.textContent = d.toLocaleString();
  }
});
