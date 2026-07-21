--TEST--
Strings 001: common string functions, sprintf, heredoc, nowdoc (NEW in 5.3)
--FILE--
<?php
var_dump(strlen("hello"));
var_dump(strtoupper("hello"));
var_dump(strtolower("HELLO"));
var_dump(trim("  hi  "));
var_dump(str_replace("world", "there", "hello world"));
var_dump(substr("hello world", 6));
var_dump(substr("hello world", 0, 5));
var_dump(strpos("hello world", "world"));
var_dump(str_repeat("ab", 3));
var_dump(implode(",", array(1, 2, 3)));
var_dump(explode(",", "a,b,c"));
var_dump(sprintf("%05d|%.2f|%s|%x", 42, 3.14159, "hi", 255));
var_dump(number_format(1234567.891, 2));
var_dump(str_pad("5", 3, "0", STR_PAD_LEFT));
var_dump(wordwrap("The quick brown fox", 10, "\n", true));

$name = "World";
$heredoc = <<<EOT
Hello, $name!
Line two.
EOT;
var_dump($heredoc);

// Nowdoc - NEW in PHP 5.3.0. No variable interpolation.
$nowdoc = <<<'EOT'
Hello, $name!
Literal, no interpolation.
EOT;
var_dump($nowdoc);
?>
--EXPECT--
int(5)
string(5) "HELLO"
string(5) "hello"
string(2) "hi"
string(11) "hello there"
string(5) "world"
string(5) "hello"
int(6)
string(6) "ababab"
string(5) "1,2,3"
array(3) {
  [0]=>
  string(1) "a"
  [1]=>
  string(1) "b"
  [2]=>
  string(1) "c"
}
string(16) "00042|3.14|hi|ff"
string(12) "1,234,567.89"
string(3) "005"
string(19) "The quick
brown fox"
string(23) "Hello, World!
Line two."
string(40) "Hello, $name!
Literal, no interpolation."
