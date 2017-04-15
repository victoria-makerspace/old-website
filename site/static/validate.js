var highlight_error = function(elem) {
    var type;
    if (elem.value) type = "warning";
    else type = "danger";
    $(elem).parents(".form-group").addClass("has-" + type);
};
var highlight = function(elem) {
    var type;
    if (!elem.validity.valid) {
        if (elem.value) type = "warning";
        else type = "danger";
    } else if (elem.value) type = "success";
    if (type) $(elem).parents(".form-group").addClass("has-" + type);
}
var clear_highlight = function(elem) {
    $(elem).parents(".form-group").removeClass("has-danger has-warning has-success");
};

var show_message = function(elem, msg) {
    $(elem).parents(".form-group").find(".form-control-feedback").text(msg);
};
var set_message = function(elem, msg) {
    elem.setCustomValidity(msg);
    show_message(elem, msg);
}
var unset_message = function(elem) {
    $(elem).parents(".form-group").find(".form-control-feedback").text("");
    elem.setCustomValidity("");
}
