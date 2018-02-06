
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

$("#membership-registration input[name=rate]").change(function() {
    if ($("#student-rate").prop("checked")) {
        $("#student").addClass("show").prop("disabled", false);
        $("#institution").focus();
    } else {
        $("#student").removeClass("show").prop("disabled", true);
    }
});
