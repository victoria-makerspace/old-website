
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

var error_class = function(elem) {
    if (!elem.validity.valid) {
        if ($(elem).val()) return "warning";
        else return "danger";
    } else if ($(elem).val()) return "success";
};
var display_error = function(elem, msg = message(elem), type = error_class(elem)) {
    highlight(type, elem);
    show_message(elem, msg);
};
var message = function(elem) {
	var msg;
    var id = $(elem).attr("id");
    if (elem.validity.valueMissing) {
        if ($(elem).is("#join input, #sign-in input")) {
            if (id == "name") msg = "Name cannot be blank.";
            if ($(elem).is("[name='username']"))
				msg = "Username cannot be blank.";
            if (id == "email") msg = "E-mail address cannot be blank.";
            if ($(elem).is("[name='password']"))
				msg = "Password cannot be blank.";
        } else if ($(elem).is("#billing input")) {
            if (id == "institution") msg = "Institution name is a required field for student members.";
            if (id == "graduation") msg = "Please enter a valid graduation date.";
        } else if ($(elem).is("#credit-card input")) {
            if (id == "name") msg = "Card holder name cannot be blank.";
        }
	} else if (elem.validity.typeMismatch) {
		if (id == "email") msg = "Invalid e-mail address.";
	} else if (elem.validity.tooShort) {
            if ($(elem).is("[name='username']"))
				msg = "Username must be at least 3 characters.";
    }
	return msg;
};
var taken = function(elem, msg) {
    exists(elem, $(elem).attr("id"), function(data) {
        if (data) {
            elem.setCustomValidity(msg);
			display_error(elem, msg);
        } else elem.setCustomValidity("");
		display_error(elem);
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
    if ($(elem).is("#sign-in [name='username']") && elem.validity.valid) {
		exists(elem, "username", function(data) {
			if (!data) {
				msg = "Username does not exist.";
				elem.setCustomValidity(msg);
				display_error(elem, msg);
			} else {
				display_error(elem);
			}
		});
	} else if ($(elem).is("#join #username")) {
        taken(elem, "Username is already taken.");
	} else if ($(elem).is("#join #email")){
		taken(elem, "E-mail address is already in use.");
    } else if (!($(elem).is("#sign-in [name='password']"))) {
		display_error(elem);
	}
};
var forms = $("#join, #billing, #credit-card, #sign-in form");
var form_control = forms.find(".form-control");
var form_submit = forms.find("[type='submit']");
form_control.focus(function() { clear_highlight(this); });
form_control.blur(function() { validate(this); });
form_submit.click(function(event) {
    var form = $(this).closest("form");
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
	if (form.is("#sign-in form")) {
		event.preventDefault();
		$.ajax("/sign-in.json", {
			data: $("#sign-in form").serialize(),
			dataType: "json",
			method: "POST",
			success: function(data) {
				if (data == "success") {
					$(location).attr("href", "/member");
				} else if (data == "invalid username") {
					$("#sign-in [name='username']").focus();
					validate($("#sign-in [name='username']")[0]);
				} else if (data == "incorrect password") {
					clear_highlight($("#sign-in [name='username']")[0]);
					highlight("success", $("#sign-in [name='username']"));
					$("#sign-in [name='password']").val("").focus();
					display_error($("#sign-in [name='password']"), "Incorrect password.", "danger");
				}
			},
			error: function(j, status, error) {
				submit = true;
				$("#sign-in form").submit();
			}
		});
	} else if (form.is("#billing") && invalid) event.preventDefault();
	else if (form.is("#join")) {
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
