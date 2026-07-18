--TEST--
P0 PHP 5.3 string increment and decrement edge semantics
--FILE--
<?php
$hex = '0x9';
++$hex;
echo ($hex === 10 ? 'hex-inc' : 'bad');
echo "\n";
$hex = '0x9';
--$hex;
echo ($hex === 8 ? 'hex-dec' : 'bad');
echo "\n";
$trail = '9 ';
++$trail;
echo ($trail === '9 ' ? 'trail-inc' : 'bad');
echo "\n";
--$trail;
echo ($trail === '9 ' ? 'trail-dec' : 'bad');
echo "\n";
$lead = ' 9';
++$lead;
echo ($lead === 10 ? 'lead' : 'bad');
echo "\n";
$carry = 'Z9';
++$carry;
echo $carry;
echo "\n";
$mixed = '9z';
++$mixed;
echo $mixed;
echo "\n";
$punct = 'a-';
++$punct;
echo ($punct === 'a-' ? 'punct' : 'bad');
?>
--EXPECT--
hex-inc
hex-dec
trail-inc
trail-dec
lead
AA0
10a
punct
