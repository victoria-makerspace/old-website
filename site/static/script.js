
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
	if ($("#shop-features").length) {
		$("body").scrollspy({ target: "#navbar-guest" });
	}
	var username = $("#sign-out button[name='sign-out']").val();
    $.getJSON("/talk/notifications.json", function(data) {
        $.each(data["notifications"], function(i, v) {
			//$("#member-menu-toolbar").after("<li>" + v["data"]["topic_title"] + "</li>");
        });
    });
});

$("#sign-in").on("shown.bs.modal", function() {
    $("#sign-in [name='username']").focus();
});
$(this).on("beanstream_payfields_loaded", function() {
    $("#credit-card input[data-beanstream-id]").each(function() {
        $(this).addClass("form-control");
        $(this).attr("id", $(this).attr("data-beanstream-id"))
    });
});
