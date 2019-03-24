function addOption() {
    var numOptions = document.getElementById("numOptions");
    var current = parseInt(numOptions.value);
    current++;
    document.getElementById("options").insertAdjacentHTML("beforeend", 
        "<input type=\"text\" name=\"option-" + current + "\"/><br>");
    numOptions.value = current;
}

// preSubmit ensures data on new election is populated
function preSubmit() {
    if (document.getElementById("newelection") != null) {
        var frm = document.getElementById("form");
        var title = frm.children[1].value;
        var startdate = frm.children[3].value;
        var enddate = frm.children[5].value;
        var opts = frm.children[8];
        if (title == "" || startdate == "" || enddate == "") {
            return false;
        }
        
        for (var i = 0; true; i+=2) {
            if (typeof opts.children[i] === "undefined") {
                break;
            }
            if (opts.children[i].value == "") {
                return false;
            }
        }
    }
}