
$("#ms-menu").on("show.bs.collapse", function() {
    $("#ms-menu-toggler").addClass("active");
});
$("#ms-menu").on("hide.bs.collapse", function() {
    $("#ms-menu-toggler").removeClass("active");
});
$("#navbar-guest .navbar-collapse").on("shown.bs.collapse", function() {
    $(".navbar-toggler").addClass("active");
});
$("#navbar-guest .navbar-collapse").on("hidden.bs.collapse", function() {
    $(".navbar-toggler").removeClass("active");
});

$(document).ready(function() {
    if (!$("html").hasClass("anon")) {
        var talk_url = $("#talk-link").attr("href");
        $.ajax(talk_url + "/session/current.json").fail(function() {
            var return_path = talk_url + "/session/current.json";
            $.ajax(talk_url + "/session/sso?return_path=" + encodeURIComponent(return_path));
        });
    }
    $(".modal:target").modal("show");
    $(".modal").on("shown.bs.modal", function() {
        $(this).find("[autofocus]").focus();
    });
});

$(this).on("beanstream_payfields_loaded", function() {
    $("#credit-card input[data-beanstream-id]").each(function() {
        $(this).addClass("form-control");
        $(this).attr("id", $(this).attr("data-beanstream-id"))
    });
});
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
    } else {
        $("#institution").focus();
    }
});
