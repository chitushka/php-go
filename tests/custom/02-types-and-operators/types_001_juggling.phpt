--TEST--
Types 001: type juggling in comparisons and arithmetic (PHP 5.3 semantics)
--FILE--
<?php
error_reporting(E_ALL & ~E_NOTICE & ~E_DEPRECATED & ~E_WARNING);
// NOTE: PHP 5.3-era loose comparison rules. Non-numeric strings compared to
// numbers convert the string to 0. This changed in PHP 8 (see RFC "saner
// string to number comparisons"), so this test is version-sensitive by design.
var_dump(0 == "abc");        // bool(true) in PHP 5.3, bool(false) in PHP 8+
var_dump("1" == "01");       // bool(true) - both numeric strings, compared as numbers
var_dump("10" == "1e1");     // bool(true) - numeric string vs scientific notation
var_dump(100 == "1e2");      // bool(true)
var_dump("abc" == 0);        // bool(true) in PHP 5.3
var_dump(null == false);     // bool(true)
var_dump(null == 0);         // bool(true)
var_dump(null == "");        // bool(true)
var_dump("0" == false);      // bool(true)
var_dump("" == false);       // bool(true)
var_dump("0.0" == "0");      // bool(true) numeric compare
var_dump(1 + "10 apples");   // int(11), with a notice suppressed here for determinism
var_dump("5" + "3");         // int(8)
var_dump("5" . "3");         // string(2) "53"
var_dump((int) "42abc");     // int(42)
var_dump((int) "abc");       // int(0)
var_dump((bool) "0");        // bool(false)
var_dump((bool) "0.0");      // bool(true)  -- "0.0" is non-empty, non-"0" string => true
var_dump((bool) array());    // bool(false)
var_dump((bool) array(0));   // bool(true)
?>
--EXPECT--
bool(true)
bool(true)
bool(true)
bool(true)
bool(true)
bool(true)
bool(true)
bool(true)
bool(true)
bool(true)
bool(true)
int(11)
int(8)
string(2) "53"
int(42)
int(0)
bool(false)
bool(true)
bool(false)
bool(true)
