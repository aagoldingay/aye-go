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

function register(func) {
        var partSize = 5;
        var code = generateID(3, partSize);
        document.getElementById("form").
            innerHTML = "<p>Sign up if the code below matches the integration</p>" 
                + "<p>" + code + "</p>" 
                + "Username (remember this): <input type=\"text\" name=\"username\" value=\"" + generateShortID(8) + "\" disabled/><br>"
                + "Password: <input type=\"password\" name=\"password\"/><br>"
                + "Confirm Password: <input type=\"password\" name=\"confirmpassword\"/></br>"
                + "<input type=\"hidden\" name=\"method\" value=\"register\"/>"
                + "<input type=\"submit\" value=\"Submit\"/>";
}