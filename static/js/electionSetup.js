function addOption() {
    var numOptions = document.getElementById("numOptions");
    var current = parseInt(numOptions.value);
    current++;
    document.getElementById("options").insertAdjacentHTML("beforeend", 
        "<input type=\"text\" name=\"option-" + current + "\"/><br>");
    numOptions.value = current;
}