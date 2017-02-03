
$("#member-menu").on("show.bs.collapse", function() {
    $("#general-menu.show").collapse("hide");
    $("#member-icon").addClass("active");
});
$("#member-menu").on("hide.bs.collapse", function() {
    $("#member-icon").removeClass("active");
});
$("#general-menu").on("show.bs.collapse", function() {
    $("#member-menu.show").collapse("hide");
    $("#general-toggler").addClass("active");
});
$("#general-menu").on("hide.bs.collapse", function() {
    $("#general-toggler").removeClass("active");
});
$("#navbar-guest .navbar-collapse").on("shown.bs.collapse", function() {
    $(".navbar-toggler").addClass("active");
});
$("#navbar-guest .navbar-collapse").on("hidden.bs.collapse", function() {
    $(".navbar-toggler").removeClass("active");
});


$(document).ready(function() {
	var username = $("#sign-out button[name='sign-out']").val();
    $.getJSON("/talk/notifications.json", function(data) {
        $.each(data["notifications"], function(i, v) {
			//$("#member-menu-toolbar").after("<li>" + v["data"]["topic_title"] + "</li>");
        });
    });
});

if ($("#shop-features").length) {
    $("body").scrollspy({ target: "#navbar-guest" });
}

$(this).on("beanstream_payfields_loaded", function() {
    $("#credit-card input[data-beanstream-id]").each(function() {
        $(this).addClass("form-control");
        $(this).attr("id", $(this).attr("data-beanstream-id"))
    });
});
$(this).on("beanstream_payfields_inputValidityChanged", function(e) {
    var args = e.originalEvent.eventDetail;
    var elem;
    if (args.fieldType == "number")
        elem = $("input[data-beanstream-id='ccNumber']");
    if (args.fieldType == "expiry")
        elem = $("input[data-beanstream-id='ccExp']");
    if (args.fieldType == "cvv")
        elem = $("input[data-beanstream-id='ccCvv']");
    if (args.isValid) {
        $(elem).parents(".form-group").removeClass("has-warning has-success").find(".form-control-feedback").hide();
        $(elem).removeClass("form-control-warning form-control-success");
        if (!elem.is(":focus")) {
            highlight("success", elem);
        }
    } else {
        $(elem).parents(".form-group").removeClass("has-success");
        $(elem).removeClass("form-control-success");
        highlight("warning", elem);
        elem.parents(".form-group").find(".form-control-feedback").show();
    }
});

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
var exists = function(elem, name, callback) { $.getJSON("/join?exists&" + name + "=" + $(elem).val()).done(callback); };

var username = $("#sign-in form [name=username]")[0];
var password = $("#sign-in form [name=password]")[0];
$("#sign-in").on("shown.bs.modal", function() {
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
$("#sign-in form").submit(function(event) {
    if (!submit) event.preventDefault();
    if (invalid_username) {
        show_message(username, "Username does not exist.");
        username.focus();
        return;
    }
    $.ajax("/sign-in.json", {
        data: $("#sign-in form").serialize(),
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
                $(location).attr("href", "/member");
            }
        },
        error: function(j, status, error) {
            submit = true;
            $("#sign-in form").submit();
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
        if ($(elem).is("#join input")) {
            if (id == "name") return "Name cannot be blank.";
            if (id == "username") return "Username cannot be blank.";
            if (id == "email") return "E-mail address cannot be blank.";
            if (id == "password") return "Password cannot be blank.";
        } else if ($(elem).is("#billing input")) {
            if (id == "institution") return "Institution name is a required field for student members.";
            if (id == "graduation") return "Please enter a valid graduation date.";
        } else if ($(elem).is("#credit-card input")) {
            if (id == "name") return "Card holder name cannot be blank.";
        }
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
$("#billing input[name=rate]").change(function() {
    var checked = $("#student-rate").prop("checked");
    var input = $("#student input");
    $("#student").prop("disabled", !checked);
    $("#student").toggleClass("text-muted", !checked);
    input.prop("required", checked);
    if (!checked) {
        $("#student .form-group").removeClass("has-danger has-success").find(".form-control-feedback").text("").hide();
        input.removeClass("form-control-danger form-control-success")
        if (!input.attr("value")) input.val("");
    }
});
var validate = function(elem) {
    msg = message(elem);
    if ($(elem).is("#join #username"))
        taken(elem, msg, "Username is already taken.");
    else if ($(elem).is("#join #email")) {
        if (elem.validity.typeMismatch) display_error(elem, "Invalid e-mail address.");
        else taken(elem, msg, "E-mail address is already in use.");
    } else display_error(elem, msg);
};
var form_control = $("#join .form-control, #billing .form-control, #credit-card .form-control");
var form_submit = $("#join [type=submit], #billing [type=submit], #credit-card [type=submit]");
form_control.focus(function() { clear_highlight(this); });
form_control.blur(function() { validate(this); });
form_submit.click(function(event) {
    var form = $(this).closest("form");
    console.log(form);
    var control = form.find(".form_control")
    control.off("focus blur");
    control.change(function() {
        clear_highlight(this);
        validate(this);
    });
    if ($(this).is(":focus")) validate(this);
    var invalid = false;
    control.each(function() {
        if (!invalid) {
            if (!this.validity.valid) {
                $(this).focus();
                invalid = true;
            }
        }
    });
    if (form.is("#billing") && invalid) event.preventDefault();
    if (form.is("#join")) {
        event.preventDefault();
        if (!invalid) {
            $.ajax("/join", {
                data: $("#join").serialize() + "&join=true",
                dataType: "json",
                method: "POST",
                success: function(data) {
                    if (data == "success") $(location).attr("href", "/member");
                },
                error: function(j, status, error) {
                    $("#join").submit();
                }
            });
        }
    }
});
