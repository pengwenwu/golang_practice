window.onload=function(){
    var userAgent = navigator.userAgent;
    console.log('***************************浏览器版本***********************\n', userAgent, '\n******************************END**************************');

    // 如果是IE浏览器，直接提示。如果不是chrome、firefox、safari，也提示
    if (doIECheck() || (!doChromeCheck() && !doFirefoxCheck() && !doSafariCheck())) {
        window.location = '/system/browsersTip';
    }

    function doIECheck() {
        var isIE = (userAgent.indexOf("compatible") > -1 && userAgent.indexOf("MSIE") > -1) ||
            userAgent.indexOf("Edge") > -1 ||
            userAgent.indexOf("Trident") > -1; //判断是否IE浏览器[低版本]
        if (isIE) {
            return true;
        }

        // IE 10以上
        if (!!window.ActiveXObject || "ActiveXObject" in window) {
            return true;
        } else {
            return false;
        }
    }

    function doChromeCheck() {
        return userAgent.indexOf("Chrome") > -1 || userAgent.indexOf("WebKit") > -1
    }

    function doFirefoxCheck() {
        return userAgent.indexOf("Firefox") > -1
    }

    function doSafariCheck() {
        return userAgent.indexOf("Safari") > -1
    }
}
