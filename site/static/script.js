
$(".navbar-collapse").on("show.bs.collapse", function() {
    $(".navbar-toggler").addClass("active");
});
$(".navbar-collapse").on("hidden.bs.collapse", function() {
    $(".navbar-toggler").removeClass("active");
});

if ($("#shop-features").length) {
    $("body").scrollspy({ target: "#navbar-guest" });
}

var highlight = function(type, elem) {
    $(elem).addClass("form-control-" + type).parents(".form-group").addClass("has-" + type);
};
var show_message = function(elem, msg) {
    if (msg) $(elem).parents(".form-group").find(".form-control-feedback").text(msg).show();
};
var clear_highlight = function(elem) {
    $(elem).parents(".form-group").removeClass("has-danger has-warning has-success").find(".form-control-feedback").text("").hide();
    $(elem).removeClass("form-control-danger form-control-warning form-control-success");
    elem.setCustomValidity("");
};
var exists = function(elem, name, callback) { $.getJSON("/exists?" + name + "=" + $(elem).val()).done(callback); };

var username = $("#signin form [name=username]")[0];
var password = $("#signin form [name=password]")[0];
$("#signin").on("shown.bs.modal", function() {
    $(username).focus();
});
var invalid_username;
$(username).change(function() {
    invalid_username = false;
    clear_highlight(username);
    exists(username, "username", function(data) {
        if (!data) {
            invalid_username = true;
            highlight("warning", username);
        } else { highlight("success", username); }
    });
});
$(password).change(function() { clear_highlight(password) });
var submit = false;
$("#signin form").submit(function(event) {
    if (!submit) event.preventDefault();
    if (invalid_username) {
        show_message(username, "Username does not exist.");
        username.focus();
        return;
    }
    $.ajax("/signin.json", {
        data: $("#signin form").serialize(),
        dataType: "json",
        method: "POST",
        success: function(data) {
            switch (data) {
            case "invalid username":
                highlight("warning", username);
                show_message(username, "Username does not exist.");
                submit = false;
                break;
            case "incorrect password":
                $(password).val("").focus();
                highlight("danger", password);
                show_message(password, "Incorrect password.");
                submit = false;
                break;
            case "success":
                $(location).attr("href", "/");
            }
        },
        error: function(j, status, error) {
            submit = true;
            $("#signin form").submit();
        }
    });
});

var error_class = function(elem) {
    if (!elem.validity.valid) {
        if ($(elem).val()) return "warning";
        else return "danger";
    } else if ($(elem).val()) return "success";
};
var display_error = function(elem, msg, type = error_class(elem)) {
    highlight(type, elem);
    show_message(elem, msg);
};
var message = function(elem) {
    var id = $(elem).attr("id");
    if (elem.validity.valueMissing) {
        if (id == "username") return "Username cannot be blank.";
        if (id == "email") return "E-mail address cannot be blank.";
        if (id == "password") return "Password cannot be blank.";
        if (id == "institution") return "Institution name is a required field for student members.";
        if (id == "graduation") return "Please enter a valid graduation date.";
    }
};
var taken = function(elem, msg, taken_msg) {
    exists(elem, $(elem).attr("id"), function(data) {
        if (data) {
            elem.setCustomValidity(taken_msg);
            msg = taken_msg;
        } else elem.setCustomValidity("");
        display_error(elem, msg);
    });
};
$("#join input[name=rate]").change(function() {
    var checked = $("#student-rate").prop("checked");
    $("#student").prop("disabled", !checked);
    $("#student").toggleClass("text-muted", !checked);
    $("#student input").prop("required", checked);
    if (!checked) {
        $("#student .form-group").removeClass("has-danger has-success").find(".form-control-feedback").text("").hide();
        $("#student .form-control").removeClass("form-control-danger form-control-success").val("");
    }
});
var validate = function(elem) {
    msg = message(elem);
    if ($(elem).is("#username"))
        taken(elem, msg, "Username is already taken.");
    else if ($(elem).is("#email")) {
        if (elem.validity.typeMismatch) display_error(elem, "Invalid e-mail address.");
        else taken(elem, msg, "E-mail address is already in use.");
    } else display_error(elem, msg);
};
$("#join .form-control").focus(function() { clear_highlight(this); });
$("#join .form-control").blur(function() { validate(this); });
$("#join [type=submit]").click(function(event) {
    $("#join .form-control").off("focus blur");
    $("#join .form-control").change(function() {
        clear_highlight(this);
        validate(this);
    });
    if ($(document.activeElement).is("#join .form-control"))
        validate(document.activeElement);
    var invalid = false;
    $("#join .form-control").each(function() {
        if (!this.validity.valid) {
            if (invalid) return;
            this.focus();
            invalid = true;
        }
    });
    if (invalid) event.preventDefault();
});
