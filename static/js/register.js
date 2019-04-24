// generate identifier to model integration
// args: parts = divisions in "XXXXX-XXXXX-XXXXX" | size = num.chars per division
function generateID(parts, size) {
    var code = "";
    var possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";

    for (var i = 0; i < parts; i++) {
        for (var j = 0; j < size; j++) {
            code += possible.charAt(Math.floor(Math.random() * possible.length));
        }
        if (i < parts-1) {
            code += "-"
        }
    }
    return code;
}

// generate shortid: acts as username
// args: size = length
function generateShortID(size) {
    var possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
    var shortID = "";
    for (i = 0; i < size; i++) {
        shortID += possible.charAt(Math.floor(Math.random() * possible.length));
    }
    return shortID;
}

function register() {
    //var partSize = 5;
    //var code = generateID(3, partSize);
    var frm = document.getElementById("form")
    frm.setAttribute("onsubmit", "return preSubmit();")
    frm.innerHTML = "<div class=\"apikey\">"
        + "Integration ID:  <input type=\"text\" name=\"part-1\" maxlength=\"5\" size=\"5\" required> - <input type=\"text\" name=\"part-2\" maxlength=\"5\" size=\"5\" required> - <input type=\"text\" name=\"part-1\" maxlength=\"5\" size=\"5\" required>"
        + "</div>" 
        + "Username (remember this): <input type=\"text\" name=\"username\" value=\"" + generateShortID(8) + "\" readonly/><br>"
        + "Password: <input type=\"password\" name=\"password\"/><br>"
        + "Confirm Password: <input type=\"password\" name=\"confirmpassword\"/><br>"
        + "Safeword: <input type=\"password\" name=\"safeword\"/> (used to prove you were not coerced when voting)</br>"
        + "Confirm Safeword: <input type=\"password\" name=\"confirmsafeword\"/><br>"
        + "<input type=\"hidden\" name=\"method\" value=\"register\"/>"
        + "<input type=\"submit\" value=\"Submit\"/>";
}

// preSubmit is only ever called as a result of registering
function preSubmit() {
    var frm = document.getElementById("form");
    var pass = frm.children[4].value;
    var confpass = frm.children[6].value;
    var sw = frm.children[8].value;
    var csw = frm.children[10].value;
    if ((pass == "" || confpass == "" || sw == "" || csw == "") || pass != confpass || sw != csw) {
        return false;
    }
    return true;
}